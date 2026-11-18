package local

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/metadata"
)

var localRevisionIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

type activationPointer struct {
	ContractVersion string `json:"contractVersion"`
	Revision        string `json:"revision"`
	MetadataDigest  string `json:"metadataDigest"`
	ActivatedAt     string `json:"activatedAt"`
}

const ActivationContractVersion = "mimir.activation/v1"

type ActivationResult struct {
	Revision string `json:"revision"`
	Status   string `json:"status"`
	Digest   string `json:"digest"`
}

type RevisionSummary struct {
	Revision string `json:"revision"`
	Status   string `json:"status"`
	Digest   string `json:"digest"`
}

// Author validates and stores an immutable non-executable draft.
func Author(ctx context.Context, candidatePath, stateDir, actor string, allowLocal bool) (RevisionSummary, error) {
	if err := ctx.Err(); err != nil {
		return RevisionSummary{}, domain.ContextFailure("metadata.author", err)
	}
	if !allowLocal || actor != "local-user" {
		if err := recordDenial(stateDir, actor, "metadata.author", "access_denied"); err != nil {
			return RevisionSummary{}, domain.NewFailure(domain.FailureUnavailable, "audit.denial", "authoring was denied and denial audit could not be persisted")
		}
		return RevisionSummary{}, domain.NewFailure(domain.FailureAccessDenied, "metadata.author", "explicit local-user authorization is required")
	}
	if candidatePath == "" || stateDir == "" {
		return RevisionSummary{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.author", "candidate and state directory are required")
	}
	file, err := os.Open(candidatePath)
	if err != nil {
		return RevisionSummary{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.load", "candidate metadata is unavailable")
	}
	raw, readErr := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return RevisionSummary{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.load", "candidate metadata could not be read")
	}
	report, err := ValidateMetadata(ctx, raw, stateDir)
	if err != nil {
		return RevisionSummary{}, err
	}
	if !report.Valid {
		return RevisionSummary{}, domain.NewFailure(
			domain.FailureInvalidInput, "metadata.author",
			"candidate metadata failed validation; report input digest is "+report.InputDigest,
		)
	}
	parsed, parseErr := metadata.ParseBytes(raw)
	if parseErr != nil {
		return RevisionSummary{}, parseErr
	}
	if parsed.Document.Status != "draft" {
		return RevisionSummary{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.author", "only valid draft metadata can be authored")
	}
	if err := (fileMetadataRepository{stateDir: stateDir}).AppendDraft(ctx, parsed.Canonical, actor); err != nil {
		return RevisionSummary{}, err
	}
	return RevisionSummary{Revision: parsed.Document.Revision, Status: parsed.Document.Status, Digest: digest(parsed.Canonical)}, nil
}

// ListMetadata returns immutable draft and activated revision identities.
func ListMetadata(stateDir, actor string, allowLocal bool) ([]RevisionSummary, error) {
	if !allowLocal || actor != "local-user" {
		if err := recordDenial(stateDir, actor, "metadata.list", "access_denied"); err != nil {
			return nil, domain.NewFailure(domain.FailureUnavailable, "audit.denial", "metadata listing was denied and denial audit could not be persisted")
		}
		return nil, domain.NewFailure(domain.FailureAccessDenied, "metadata.list", "explicit local-user authorization is required")
	}
	if !isPrivateDirectory(stateDir) {
		return nil, domain.NewFailure(domain.FailureInvalidInput, "metadata.list", "private local metadata directory is unavailable")
	}
	revisions, err := (fileMetadataRepository{stateDir: stateDir}).List(context.Background())
	if err != nil {
		return nil, err
	}
	summaries := make([]RevisionSummary, 0, len(revisions))
	for _, raw := range revisions {
		parsed, err := metadata.Parse(bytes.NewReader(raw))
		if err != nil {
			return nil, domain.NewFailure(domain.FailureInvalidInput, "metadata.list", "stored metadata revision failed validation")
		}
		summaries = append(summaries, RevisionSummary{
			Revision: parsed.Document.Revision, Status: parsed.Document.Status, Digest: digest(raw),
		})
	}
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Revision == summaries[j].Revision {
			return summaries[i].Status < summaries[j].Status
		}
		return summaries[i].Revision < summaries[j].Revision
	})
	return summaries, nil
}

// ActivateDraft activates a previously authored immutable draft by revision ID.
func ActivateDraft(ctx context.Context, revision, stateDir, actor string, allowLocal bool, expectedActive string) (ActivationResult, error) {
	if !localRevisionIDPattern.MatchString(revision) {
		return ActivationResult{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.activate", "revision identifier is invalid")
	}
	path := filepath.Join(stateDir, "metadata", "drafts", revision+".json")
	return Activate(ctx, path, stateDir, actor, allowLocal, expectedActive)
}

// Activate validates and immutably stores a candidate before conditionally
// replacing the active pointer. The local policy is explicit and non-production.
func Activate(ctx context.Context, candidatePath, stateDir, actor string, allowLocal bool, expectedActive ...string) (ActivationResult, error) {
	if err := ctx.Err(); err != nil {
		return ActivationResult{}, domain.ContextFailure("metadata.activate", err)
	}
	if !allowLocal || actor != "local-user" {
		if err := recordDenial(stateDir, actor, "metadata.activate", "access_denied"); err != nil {
			return ActivationResult{}, domain.NewFailure(domain.FailureUnavailable, "audit.denial", "activation was denied and denial audit could not be persisted")
		}
		return ActivationResult{}, domain.NewFailure(domain.FailureAccessDenied, "metadata.activate", "explicit local-user authorization is required")
	}
	if candidatePath == "" || stateDir == "" {
		return ActivationResult{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.activate", "candidate and state directory are required")
	}
	candidate, err := os.Open(candidatePath)
	if err != nil {
		return ActivationResult{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.load", "candidate metadata is unavailable")
	}
	parsed, parseErr := metadata.Parse(candidate)
	closeErr := candidate.Close()
	if parseErr != nil {
		return ActivationResult{}, parseErr
	}
	if closeErr != nil {
		return ActivationResult{}, domain.NewFailure(domain.FailureUnavailable, "metadata.load", "candidate metadata could not be closed")
	}
	if parsed.Document.Status != "draft" {
		return ActivationResult{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.activate", "only draft metadata can be activated")
	}
	if err := localMetadataPolicy(parsed.Document); err != nil {
		if err := recordDenial(stateDir, actor, "metadata.activate", "access_denied"); err != nil {
			return ActivationResult{}, domain.NewFailure(domain.FailureUnavailable, "audit.denial", "activation was denied and denial audit could not be persisted")
		}
		return ActivationResult{}, domain.NewFailure(domain.FailureAccessDenied, "metadata.activate", "local reference profile permits only public or internal metadata")
	}
	if err := validateLocalMetadataCapabilities(parsed.Document); err != nil {
		return ActivationResult{}, err
	}
	parsed.Document.Status = "active"
	activeBytes, err := json.Marshal(parsed.Document)
	if err != nil {
		return ActivationResult{}, domain.NewFailure(domain.FailureInternal, "metadata.activate", "active revision cannot be encoded")
	}
	metadataDigest := digest(activeBytes)
	if len(expectedActive) > 1 {
		return ActivationResult{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.activate", "only one expected active revision may be supplied")
	}
	expected := ""
	if len(expectedActive) == 1 {
		expected = expectedActive[0]
	}
	previousRevision, err := (fileMetadataRepository{stateDir: stateDir}).Activate(ctx, activeBytes, expected)
	if err != nil {
		return ActivationResult{}, err
	}
	auditPath := filepath.Join(stateDir, "audit", "events.jsonl")
	eventID := digest([]byte("metadata.activate\x00" + parsed.Document.Revision + "\x00" + metadataDigest))
	event := auditEvent{
		EventID: eventID, Operation: "metadata.activated", Actor: actor,
		MetadataRevision: parsed.Document.Revision, PreviousRevision: previousRevision,
		OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := ensureAudit(auditPath, event); err != nil {
		return ActivationResult{}, domain.NewFailure(domain.FailureUnavailable, "audit.append", "activation is stored but its required audit event is pending; retry activation")
	}
	return ActivationResult{Revision: parsed.Document.Revision, Status: parsed.Document.Status, Digest: metadataDigest}, nil
}

func loadActiveMetadata(stateDir string) (metadata.Parsed, activationPointer, error) {
	raw, err := (fileMetadataRepository{stateDir: stateDir}).LoadActive(context.Background())
	if err != nil {
		return metadata.Parsed{}, activationPointer{}, err
	}
	pointers, err := readAndRepairTail[activationPointer](filepath.Join(stateDir, "metadata", "active.jsonl"))
	if err != nil || len(pointers) == 0 {
		return metadata.Parsed{}, activationPointer{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.active", "no active metadata revision is available")
	}
	pointer := pointers[len(pointers)-1]
	if pointer.ContractVersion != ActivationContractVersion || pointer.MetadataDigest == "" || digest(raw) != pointer.MetadataDigest {
		return metadata.Parsed{}, activationPointer{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.active", "active metadata pointer has an unsupported version")
	}
	parsed, err := metadata.Parse(bytes.NewReader(raw))
	if err != nil || parsed.Document.Status != "active" || parsed.Document.Revision != pointer.Revision {
		return metadata.Parsed{}, activationPointer{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.active", "active metadata revision failed validation")
	}
	auditPath := filepath.Join(stateDir, "audit", "events.jsonl")
	if _, err := os.Stat(auditPath); err != nil {
		return metadata.Parsed{}, activationPointer{}, domain.NewFailure(domain.FailureUnavailable, "metadata.active", "activation audit evidence is unavailable")
	}
	audits, err := readAndRepairTail[auditEvent](auditPath)
	if err != nil {
		return metadata.Parsed{}, activationPointer{}, err
	}
	expectedID := digest([]byte("metadata.activate\x00" + pointer.Revision + "\x00" + pointer.MetadataDigest))
	for _, event := range audits {
		if event.EventID == expectedID && event.Operation == "metadata.activated" {
			return parsed, pointer, nil
		}
	}
	return metadata.Parsed{}, activationPointer{}, domain.NewFailure(domain.FailureUnavailable, "metadata.active", "activation audit is pending; retry activation before running")
}

func writeImmutable(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			existing, readErr := os.ReadFile(path)
			if readErr == nil && string(existing) == string(data) {
				return nil
			}
			return domain.NewFailure(domain.FailureConflict, "metadata.repository", "revision identity already has different content")
		}
		return domain.NewFailure(domain.FailureUnavailable, "metadata.repository", "revision could not be stored")
	}
	n, writeErr := file.Write(data)
	if writeErr == nil && n != len(data) {
		writeErr = io.ErrShortWrite
	}
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(path)
		return domain.NewFailure(domain.FailureUnavailable, "metadata.repository", "revision could not be durably stored")
	}
	return syncDirectory(filepath.Dir(path))
}

func ensurePrivateDirectory(path string) error {
	var missing []string
	for current := path; ; current = filepath.Dir(current) {
		_, err := os.Lstat(current)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		missing = append(missing, current)
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	for i := len(missing) - 1; i >= 0; i-- {
		directory := missing[i]
		if err := createPrivateDirectory(directory); err != nil && !os.IsExist(err) {
			return err
		}
		if err := syncDirectory(filepath.Dir(directory)); err != nil {
			return err
		}
	}
	if !isPrivateDirectory(path) {
		return fmt.Errorf("directory must be private to the local user")
	}
	return nil
}
