package local

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/metadata"
	"github.com/kumerian/mimir/internal/ports"
)

type fileMetadataRepository struct {
	stateDir string
}

func (r fileMetadataRepository) Capabilities() ports.CapabilitySet {
	return ports.CapabilitySet{
		ContractVersion: ports.MetadataRepositoryVersion,
		Operations:      []string{"append_draft", "list_revisions", "load_active", "conditional_activate"},
		Unsupported:     []string{"distributed_transactions", "automatic_migration"},
		Concurrency:     1,
		MaximumBytes:    1 << 20,
		Classifications: []string{"public", "internal"},
	}
}

func (r fileMetadataRepository) AppendDraft(ctx context.Context, raw json.RawMessage, actor string) error {
	if err := ctx.Err(); err != nil {
		return domain.ContextFailure("metadata.repository.append", err)
	}
	parsed, err := metadata.Parse(bytes.NewReader(raw))
	if err != nil {
		return err
	}
	if parsed.Document.Status != "draft" || actor == "" {
		return domain.NewFailure(domain.FailureInvalidInput, "metadata.repository.append", "draft metadata and author identity are required")
	}
	if err := ensurePrivateDirectory(r.stateDir); err != nil {
		return domain.NewFailure(domain.FailureUnavailable, "metadata.repository.append", "metadata repository directory is unavailable")
	}
	lock, err := acquireLock(r.stateDir)
	if err != nil {
		return err
	}
	defer releaseLock(lock)
	draftDir := filepath.Join(r.stateDir, "metadata", "drafts")
	if err := ensurePrivateDirectory(draftDir); err != nil {
		return domain.NewFailure(domain.FailureUnavailable, "metadata.repository.append", "draft directory is unavailable")
	}
	if err := writeImmutable(filepath.Join(draftDir, parsed.Document.Revision+".json"), parsed.Canonical); err != nil {
		return err
	}
	event := auditEvent{
		EventID:   digest([]byte("metadata.authored\x00" + parsed.Document.Revision + "\x00" + digest(parsed.Canonical))),
		Operation: "metadata.authored", Actor: actor, MetadataRevision: parsed.Document.Revision,
		OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := ensureAudit(filepath.Join(r.stateDir, "audit", "events.jsonl"), event); err != nil {
		return domain.NewFailure(domain.FailureUnavailable, "audit.append", "draft is stored but its audit event is pending; retry authoring")
	}
	return nil
}

func (r fileMetadataRepository) List(ctx context.Context) ([]json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, domain.ContextFailure("metadata.repository.list", err)
	}
	if r.stateDir == "" {
		return nil, domain.NewFailure(domain.FailureInvalidInput, "metadata.repository.list", "metadata repository directory is required")
	}
	revisions := make([]json.RawMessage, 0)
	for _, directory := range []string{
		filepath.Join(r.stateDir, "metadata", "drafts"),
		filepath.Join(r.stateDir, "metadata", "revisions"),
	} {
		entries, err := os.ReadDir(directory)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, domain.NewFailure(domain.FailureUnavailable, "metadata.repository.list", "metadata history is unavailable")
		}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
				continue
			}
			if err := ctx.Err(); err != nil {
				return nil, domain.ContextFailure("metadata.repository.list", err)
			}
			path := filepath.Join(directory, entry.Name())
			raw, err := os.ReadFile(path)
			if err != nil {
				return nil, domain.NewFailure(domain.FailureUnavailable, "metadata.repository.list", "metadata revision cannot be read")
			}
			if _, err := metadata.Parse(bytes.NewReader(raw)); err != nil {
				return nil, domain.NewFailure(domain.FailureInvalidInput, "metadata.repository.list", "stored metadata revision failed validation")
			}
			revisions = append(revisions, json.RawMessage(raw))
		}
	}
	return revisions, nil
}

func (r fileMetadataRepository) LoadActive(ctx context.Context) (json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, domain.ContextFailure("metadata.repository.active", err)
	}
	pointers, err := readAndRepairTail[activationPointer](filepath.Join(r.stateDir, "metadata", "active.jsonl"))
	if err != nil || len(pointers) == 0 {
		return nil, domain.NewFailure(domain.FailureInvalidInput, "metadata.active", "no active metadata revision is available")
	}
	pointer := pointers[len(pointers)-1]
	if pointer.ContractVersion != ActivationContractVersion || pointer.MetadataDigest == "" || pointer.Revision == "" {
		return nil, domain.NewFailure(domain.FailureInvalidInput, "metadata.active", "active metadata pointer has an unsupported version")
	}
	path := filepath.Join(r.stateDir, "metadata", "revisions", pointer.MetadataDigest+".json")
	raw, err := os.ReadFile(path)
	if err != nil || digest(raw) != pointer.MetadataDigest {
		return nil, domain.NewFailure(domain.FailureInvalidInput, "metadata.active", "active metadata revision is unavailable or has changed")
	}
	if !isPrivateFile(path) {
		return nil, domain.NewFailure(domain.FailureAccessDenied, "metadata.active", "active metadata permissions must be private to the local user")
	}
	parsed, err := metadata.Parse(bytes.NewReader(raw))
	if err != nil || parsed.Document.Status != "active" || parsed.Document.Revision != pointer.Revision {
		return nil, domain.NewFailure(domain.FailureInvalidInput, "metadata.active", "active metadata revision failed validation")
	}
	return json.RawMessage(raw), nil
}

// Activate performs a conditional immutable activation. The calling use case
// is responsible for authorization and for recording the corresponding audit
// event before exposing this revision to execution.
func (r fileMetadataRepository) Activate(ctx context.Context, raw json.RawMessage, expectedActive string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", domain.ContextFailure("metadata.repository.activate", err)
	}
	parsed, err := metadata.Parse(bytes.NewReader(raw))
	if err != nil || parsed.Document.Status != "active" || !bytes.Equal(parsed.Canonical, raw) {
		return "", domain.NewFailure(domain.FailureInvalidInput, "metadata.repository.activate", "canonical active metadata is required")
	}
	if err := ensurePrivateDirectory(r.stateDir); err != nil {
		return "", domain.NewFailure(domain.FailureUnavailable, "metadata.repository.activate", "metadata repository directory is unavailable")
	}
	lock, err := acquireLock(r.stateDir)
	if err != nil {
		return "", err
	}
	defer releaseLock(lock)
	pointerPath := filepath.Join(r.stateDir, "metadata", "active.jsonl")
	if err := ensurePrivateDirectory(filepath.Dir(pointerPath)); err != nil {
		return "", domain.NewFailure(domain.FailureUnavailable, "metadata.repository.activate", "metadata directory is unavailable")
	}
	pointers, err := readAndRepairTail[activationPointer](pointerPath)
	if err != nil {
		return "", err
	}
	var previous activationPointer
	if len(pointers) > 0 {
		previous = pointers[len(pointers)-1]
	}
	if previous.Revision != "" && previous.Revision != parsed.Document.Revision && expectedActive == "" {
		return "", domain.NewFailure(domain.FailureInvalidInput, "metadata.repository.activate", "expected active revision is required when superseding an active revision")
	}
	if expectedActive != "" && expectedActive != previous.Revision {
		return "", domain.NewFailure(domain.FailureConflict, "metadata.repository.activate", "active revision changed; candidate activation was not applied")
	}
	metadataDigest := digest(raw)
	revisionDir := filepath.Join(r.stateDir, "metadata", "revisions")
	if err := ensurePrivateDirectory(revisionDir); err != nil {
		return "", domain.NewFailure(domain.FailureUnavailable, "metadata.repository.activate", "revision directory is unavailable")
	}
	existingRevisions, err := os.ReadDir(revisionDir)
	if err != nil {
		return "", domain.NewFailure(domain.FailureUnavailable, "metadata.repository.activate", "revision history is unavailable")
	}
	for _, entry := range existingRevisions {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		existingBytes, readErr := os.ReadFile(filepath.Join(revisionDir, entry.Name()))
		if readErr != nil {
			return "", domain.NewFailure(domain.FailureUnavailable, "metadata.repository.activate", "revision history cannot be verified")
		}
		existing, parseErr := metadata.Parse(bytes.NewReader(existingBytes))
		if parseErr != nil {
			return "", domain.NewFailure(domain.FailureInvalidInput, "metadata.repository.activate", "stored revision failed validation")
		}
		if existing.Document.Revision == parsed.Document.Revision && digest(existingBytes) != metadataDigest {
			return "", domain.NewFailure(domain.FailureConflict, "metadata.repository.activate", "revision identity is immutable and already has different content")
		}
	}
	if err := writeImmutable(filepath.Join(revisionDir, metadataDigest+".json"), raw); err != nil {
		return "", err
	}
	pointer := activationPointer{
		ContractVersion: ActivationContractVersion, Revision: parsed.Document.Revision,
		MetadataDigest: metadataDigest, ActivatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	if previous.Revision == pointer.Revision && previous.MetadataDigest == pointer.MetadataDigest {
		return previous.Revision, nil
	}
	if err := appendSynced(pointerPath, pointer); err != nil {
		return "", domain.NewFailure(domain.FailureUnavailable, "metadata.repository.activate", "active pointer could not be durably appended")
	}
	return previous.Revision, nil
}

var _ ports.MetadataRepository = fileMetadataRepository{}
