package local

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"time"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/ports"
)

type fileAuditStore struct {
	path string
}

func (s fileAuditStore) Capabilities() ports.CapabilitySet {
	return ports.CapabilitySet{
		ContractVersion: ports.AuditStoreVersion,
		Operations:      []string{"append", "deduplicate_by_event_id"},
		Unsupported:     []string{"remote_export", "automatic_retention"},
		Concurrency:     1,
		MaximumBytes:    maxSourceBytes,
		Classifications: []string{"public", "internal"},
	}
}

func (s fileAuditStore) Append(ctx context.Context, raw json.RawMessage) error {
	if err := ctx.Err(); err != nil {
		return domain.ContextFailure("audit.append", err)
	}
	var event auditEvent
	if err := decodeStrict(raw, &event); err != nil || event.EventID == "" || event.Operation == "" ||
		event.Actor == "" || event.OccurredAt == "" {
		return domain.NewFailure(domain.FailureInvalidInput, "audit.append", "audit event is incomplete or invalid")
	}
	if !validAuditEvent(event) {
		return domain.NewFailure(domain.FailureInvalidInput, "audit.append", "audit event contains an unsupported or unbounded field")
	}
	if _, err := time.Parse(time.RFC3339Nano, event.OccurredAt); err != nil {
		return domain.NewFailure(domain.FailureInvalidInput, "audit.append", "audit event time is invalid")
	}
	if s.path == "" {
		return domain.NewFailure(domain.FailureInvalidInput, "audit.append", "audit journal path is required")
	}
	if err := ensurePrivateDirectory(filepath.Dir(s.path)); err != nil {
		return domain.NewFailure(domain.FailureUnavailable, "audit.append", "audit directory is unavailable")
	}
	lock, err := acquireJournalLock(s.path)
	if err != nil {
		return err
	}
	defer releaseLock(lock)
	if err := ctx.Err(); err != nil {
		return domain.ContextFailure("audit.append", err)
	}
	if err := appendAuditEvent(s.path, event); err != nil {
		return err
	}
	return nil
}

func validAuditEvent(event auditEvent) bool {
	operations := map[string]bool{
		"metadata.authored": true, "metadata.activated": true, "metadata.discovered": true,
		"schedule.requested": true, "schedule.stopped": true, "run.planned": true, "run.committed": true,
		"run.failed": true, "replay.prepared": true, "cleaning.applied": true,
		"access.denied": true,
	}
	if !isDigest(event.EventID) || !operations[event.Operation] ||
		(event.Actor != "local-user" && event.Actor != "unknown") ||
		(event.MetadataRevision != "" && !localRevisionIDPattern.MatchString(event.MetadataRevision)) ||
		(event.PreviousRevision != "" && !localRevisionIDPattern.MatchString(event.PreviousRevision)) ||
		(event.PlanID != "" && !localRevisionIDPattern.MatchString(event.PlanID)) ||
		(event.RunID != "" && !localRevisionIDPattern.MatchString(event.RunID)) ||
		(event.OriginalRunID != "" && !localRevisionIDPattern.MatchString(event.OriginalRunID)) ||
		(event.AttemptedOperation != "" && !localRevisionIDPattern.MatchString(event.AttemptedOperation)) ||
		(event.FailureStage != "" && !localRevisionIDPattern.MatchString(event.FailureStage)) ||
		(event.CorrectionID != "" && !localRevisionIDPattern.MatchString(event.CorrectionID)) ||
		(event.CorrectionVersion != "" && event.CorrectionVersion != "v1") ||
		(event.InputDigest != "" && !isDigest(event.InputDigest)) ||
		(event.OutputDigest != "" && !isDigest(event.OutputDigest)) ||
		event.RecordCount < 0 {
		return false
	}
	if event.FailureCategory != "" && !validFailureCategory(domain.FailureCategory(event.FailureCategory)) {
		return false
	}
	return true
}

type telemetryEvent struct {
	EventID          string `json:"eventId"`
	Name             string `json:"name"`
	Stage            string `json:"stage"`
	Outcome          string `json:"outcome"`
	FailureCategory  string `json:"failureCategory,omitempty"`
	MetadataRevision string `json:"metadataRevision"`
	PlanID           string `json:"planId"`
	RunID            string `json:"runId"`
	Attempt          int    `json:"attempt"`
	OccurredAt       string `json:"occurredAt"`
}

type fileTelemetry struct {
	path string
}

func (s fileTelemetry) Capabilities() ports.CapabilitySet {
	return ports.CapabilitySet{
		ContractVersion: ports.TelemetryVersion,
		Operations:      []string{"emit_structured_correlation"},
		Unsupported:     []string{"metrics_export", "trace_export", "arbitrary_attributes"},
		Concurrency:     1,
		MaximumBytes:    4096,
		Classifications: []string{"public", "internal"},
	}
}

func (s fileTelemetry) Emit(ctx context.Context, raw json.RawMessage) error {
	if err := ctx.Err(); err != nil {
		return domain.ContextFailure("telemetry.emit", err)
	}
	var event telemetryEvent
	if err := decodeStrict(raw, &event); err != nil || event.EventID == "" || event.Name == "" ||
		event.Stage == "" || event.Outcome == "" || event.MetadataRevision == "" ||
		event.PlanID == "" || event.RunID == "" || event.Attempt < 0 {
		return domain.NewFailure(domain.FailureInvalidInput, "telemetry.emit", "structured telemetry event is incomplete or invalid")
	}
	if !validTelemetryEvent(event) {
		return domain.NewFailure(domain.FailureInvalidInput, "telemetry.emit", "telemetry event contains an unsupported or unbounded field")
	}
	if _, err := time.Parse(time.RFC3339Nano, event.OccurredAt); err != nil {
		return domain.NewFailure(domain.FailureInvalidInput, "telemetry.emit", "telemetry event time is invalid")
	}
	if s.path == "" {
		return domain.NewFailure(domain.FailureInvalidInput, "telemetry.emit", "telemetry journal path is required")
	}
	if err := ensurePrivateDirectory(filepath.Dir(s.path)); err != nil {
		return domain.NewFailure(domain.FailureUnavailable, "telemetry.emit", "telemetry directory is unavailable")
	}
	lock, err := acquireJournalLock(s.path)
	if err != nil {
		return err
	}
	defer releaseLock(lock)
	if err := ctx.Err(); err != nil {
		return domain.ContextFailure("telemetry.emit", err)
	}
	if err := appendSynced(s.path, event); err != nil {
		return domain.NewFailure(domain.FailureUnavailable, "telemetry.emit", "local telemetry event could not be written")
	}
	return nil
}

func validTelemetryEvent(event telemetryEvent) bool {
	names := map[string]bool{
		"run.planned": true, "run.started": true, "destination.committed": true,
		"destination.reconciled": true, "checkpoint.committed": true,
		"run.committed": true, "run.succeeded": true, "run.failed": true,
	}
	stages := map[string]bool{
		"planning": true, "execution": true, "publication": true,
		"checkpoint": true, "audit": true, "completion": true,
		"run.transform": true, "run.quality": true,
	}
	outcomes := map[string]bool{
		"planned": true, "started": true, "committed": true, "failed": true, "succeeded": true,
	}
	return isDigest(event.EventID) && names[event.Name] && stages[event.Stage] &&
		outcomes[event.Outcome] &&
		localRevisionIDPattern.MatchString(event.MetadataRevision) &&
		localRevisionIDPattern.MatchString(event.PlanID) &&
		localRevisionIDPattern.MatchString(event.RunID) &&
		(event.Attempt > 0 || event.Name == "run.planned") &&
		(event.FailureCategory == "" || validFailureCategory(domain.FailureCategory(event.FailureCategory)))
}

func validFailureCategory(category domain.FailureCategory) bool {
	switch category {
	case domain.FailureInvalidInput, domain.FailureUnsupportedCapability, domain.FailureAccessDenied,
		domain.FailureConflict, domain.FailurePolicyRejected, domain.FailureUnavailable,
		domain.FailureAmbiguousCommit, domain.FailureCancelled, domain.FailureTimeout, domain.FailureInternal:
		return true
	default:
		return false
	}
}

func isDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

var _ ports.AuditStore = fileAuditStore{}
var _ ports.Telemetry = fileTelemetry{}
