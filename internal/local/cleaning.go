package local

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/kumerian/mimir/internal/cleaning"
	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/metadata"
)

type CleaningOptions struct {
	StateDir       string
	RecordPath     string
	Actor          string
	Classification string
	AllowLocal     bool
	Preview        bool
	Correction     cleaning.Correction
}

type CleaningResult struct {
	Record         map[string]any `json:"record"`
	Audit          cleaning.Audit `json:"audit"`
	Preview        bool           `json:"preview"`
	AlreadyApplied bool           `json:"alreadyApplied"`
}

type cleanedEntry struct {
	IdempotencyKey string          `json:"idempotencyKey"`
	CorrectionID   string          `json:"correctionId"`
	Version        string          `json:"correctionVersion"`
	Actor          string          `json:"actor"`
	InputReference string          `json:"inputReference"`
	OutputDigest   string          `json:"outputDigest"`
	Record         json.RawMessage `json:"record"`
}

// CleanRecord previews or applies one explicit correction without modifying
// its input. Applied records are stored separately and their audit omits data.
func CleanRecord(ctx context.Context, options CleaningOptions) (CleaningResult, error) {
	if err := ctx.Err(); err != nil {
		return CleaningResult{}, domain.ContextFailure("cleaning.start", err)
	}
	if !options.AllowLocal || options.Actor != "local-user" {
		if err := recordDenial(options.StateDir, options.Actor, "cleaning.apply", "access_denied"); err != nil {
			return CleaningResult{}, domain.NewFailure(domain.FailureUnavailable, "audit.denial", "cleaning was denied and denial audit could not be persisted")
		}
		return CleaningResult{}, domain.NewFailure(domain.FailureAccessDenied, "cleaning.apply", "explicit local-user authorization is required")
	}
	if options.RecordPath == "" {
		return CleaningResult{}, domain.NewFailure(domain.FailureInvalidInput, "cleaning.input", "record file is required")
	}
	info, err := os.Stat(options.RecordPath)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 256<<10 {
		return CleaningResult{}, domain.NewFailure(domain.FailureInvalidInput, "cleaning.input", "record must be a regular JSON object file no larger than 256 KiB")
	}
	raw, err := os.ReadFile(options.RecordPath)
	if err != nil {
		return CleaningResult{}, domain.NewFailure(domain.FailureUnavailable, "cleaning.input", "record file is unavailable")
	}
	return cleanRecordBytes(ctx, options, raw)
}

// CleanRecordBytes previews or applies one explicit correction from bounded
// inline JSON without staging the record on disk.
func CleanRecordBytes(ctx context.Context, options CleaningOptions, raw []byte) (CleaningResult, error) {
	if err := ctx.Err(); err != nil {
		return CleaningResult{}, domain.ContextFailure("cleaning.start", err)
	}
	if !options.AllowLocal || options.Actor != "local-user" {
		if err := recordDenial(options.StateDir, options.Actor, "cleaning.apply", "access_denied"); err != nil {
			return CleaningResult{}, domain.NewFailure(domain.FailureUnavailable, "audit.denial", "cleaning was denied and denial audit could not be persisted")
		}
		return CleaningResult{}, domain.NewFailure(domain.FailureAccessDenied, "cleaning.apply", "explicit local-user authorization is required")
	}
	if len(raw) == 0 || len(raw) > 256<<10 {
		return CleaningResult{}, domain.NewFailure(domain.FailureInvalidInput, "cleaning.input", "record is missing or exceeds the 256 KiB size limit")
	}
	return cleanRecordBytes(ctx, options, raw)
}

func cleanRecordBytes(ctx context.Context, options CleaningOptions, raw []byte) (CleaningResult, error) {
	if err := ctx.Err(); err != nil {
		return CleaningResult{}, domain.ContextFailure("cleaning.start", err)
	}
	if options.Classification != "public" && options.Classification != "internal" {
		return CleaningResult{}, domain.NewFailure(domain.FailureAccessDenied, "cleaning.policy", "local cleaning permits only explicitly classified public or internal records")
	}
	canonical, err := metadata.CanonicalizeRecord(raw)
	if err != nil {
		return CleaningResult{}, domain.NewFailure(domain.FailureInvalidInput, "cleaning.input", "record file is not a valid JSON object")
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.UseNumber()
	var record map[string]any
	if err := decoder.Decode(&record); err != nil {
		return CleaningResult{}, domain.NewFailure(domain.FailureInvalidInput, "cleaning.input", "record file cannot be decoded")
	}
	applied, err := cleaning.Apply(record, options.Correction, options.Actor, true)
	if err != nil {
		return CleaningResult{}, err
	}
	output, err := metadata.CanonicalJSON(applied.Record)
	if err != nil {
		return CleaningResult{}, domain.NewFailure(domain.FailureInvalidInput, "cleaning.output", "cleaned record cannot be canonicalized")
	}
	result := CleaningResult{Record: applied.Record, Audit: applied.Audit, Preview: options.Preview}
	if options.Preview {
		return result, nil
	}
	if options.StateDir == "" {
		return CleaningResult{}, domain.NewFailure(domain.FailureInvalidInput, "cleaning.state", "state directory is required to apply cleaning")
	}
	if err := ensurePrivateDirectory(options.StateDir); err != nil {
		return CleaningResult{}, domain.NewFailure(domain.FailureUnavailable, "cleaning.state", "private local state directory is unavailable")
	}
	lock, err := acquireLock(options.StateDir)
	if err != nil {
		return CleaningResult{}, err
	}
	defer releaseLock(lock)

	entry := cleanedEntry{
		IdempotencyKey: applied.Audit.IdempotencyKey, CorrectionID: applied.Audit.CorrectionID,
		Version: applied.Audit.CorrectionVersion, Actor: options.Actor,
		InputReference: applied.Audit.InputReference, OutputDigest: applied.Audit.OutputDigest,
		Record: output,
	}
	dataPath := filepath.Join(options.StateDir, "data", "cleaned.jsonl")
	if err := ensurePrivateDirectory(filepath.Dir(dataPath)); err != nil {
		return CleaningResult{}, domain.NewFailure(domain.FailureUnavailable, "cleaning.persist", "cleaned data directory is unavailable")
	}
	entries, err := readAndRepairTail[cleanedEntry](dataPath)
	if err != nil {
		return CleaningResult{}, err
	}
	found := false
	count := 0
	for _, prior := range entries {
		if prior.IdempotencyKey != entry.IdempotencyKey {
			continue
		}
		count++
		if !sameCleanedEntry(prior, entry) {
			return CleaningResult{}, domain.NewFailure(domain.FailureConflict, "cleaning.persist", "cleaning identity conflicts with persisted output")
		}
		found = true
	}
	if count > 1 {
		return CleaningResult{}, domain.NewFailure(domain.FailureConflict, "cleaning.persist", "multiple outputs exist for the same cleaning identity")
	}
	if !found {
		if err := appendSynced(dataPath, entry); err != nil {
			return CleaningResult{}, domain.NewFailure(domain.FailureAmbiguousCommit, "cleaning.persist", "cleaning output may be committed; retry to reconcile")
		}
	}
	auditPath := filepath.Join(options.StateDir, "audit", "events.jsonl")
	event := auditEvent{
		EventID: applied.Audit.IdempotencyKey, Operation: "cleaning.applied",
		Actor: options.Actor, RunID: "", PlanID: applied.Audit.CorrectionID,
		CorrectionID: applied.Audit.CorrectionID, CorrectionVersion: applied.Audit.CorrectionVersion,
		InputDigest: applied.Audit.InputReference, OutputDigest: applied.Audit.OutputDigest,
		RecordCount: 1, OccurredAt: applied.Audit.OccurredAt,
	}
	if err := ensureAudit(auditPath, event); err != nil {
		return CleaningResult{}, domain.NewFailure(domain.FailureUnavailable, "cleaning.audit", "cleaned output is stored but its audit event is pending; retry to reconcile")
	}
	result.AlreadyApplied = found
	return result, nil
}

func sameCleanedEntry(left, right cleanedEntry) bool {
	leftBytes, leftErr := json.Marshal(left)
	rightBytes, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftBytes, rightBytes)
}
