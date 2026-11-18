package local

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/ports"
	"github.com/kumerian/mimir/internal/state"
)

// fileRunStateStore is the local, single-host implementation of the versioned
// run-state port. Its exclusive journal lock serializes conditional appends;
// the event history itself provides fencing across attempts.
type fileRunStateStore struct {
	path string
}

func (s fileRunStateStore) Capabilities() ports.CapabilitySet {
	return ports.CapabilitySet{
		ContractVersion: ports.RunStateStoreVersion,
		Operations:      []string{"append_transition", "load_history", "owner_fencing"},
		Unsupported:     []string{"distributed_lease", "remote_worker_ownership"},
		Concurrency:     1,
		MaximumBytes:    maxSourceBytes,
		Classifications: []string{"public", "internal"},
	}
}

func (s fileRunStateStore) AppendTransition(ctx context.Context, runID string, raw json.RawMessage) error {
	if err := ctx.Err(); err != nil {
		return domain.ContextFailure("run.state.append", err)
	}
	if runID == "" || s.path == "" {
		return domain.NewFailure(domain.FailureInvalidInput, "run.state.append", "run identity and state path are required")
	}
	var next state.Event
	if err := decodeStrict(raw, &next); err != nil || next.RunID != runID {
		return domain.NewFailure(domain.FailureInvalidInput, "run.state.append", "run-state event is invalid")
	}
	if next.ContractVersion != state.ContractVersion {
		return domain.NewFailure(domain.FailureUnsupportedCapability, "run.state.append", "run-state contract version is unsupported")
	}
	if err := ensurePrivateDirectory(filepath.Dir(s.path)); err != nil {
		return domain.NewFailure(domain.FailureUnavailable, "run.state.append", "run-state directory is unavailable")
	}
	lock, err := acquireJournalLock(s.path)
	if err != nil {
		return err
	}
	defer releaseLock(lock)

	allEvents, err := readAndRepairTail[state.Event](s.path)
	if err != nil {
		return err
	}
	history := make([]state.Event, 0)
	for _, event := range allEvents {
		if event.RunID == runID {
			history = append(history, event)
		}
	}
	if err := validateNextStateEvent(history, next); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return domain.ContextFailure("run.state.append", err)
	}
	if err := appendSynced(s.path, next); err != nil {
		return domain.NewFailure(domain.FailureUnavailable, "run.state.append", "run-state transition could not be durably appended")
	}
	return nil
}

func (s fileRunStateStore) LoadHistory(ctx context.Context, runID string) ([]json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, domain.ContextFailure("run.state.load", err)
	}
	if runID == "" || s.path == "" {
		return nil, domain.NewFailure(domain.FailureInvalidInput, "run.state.load", "run identity and state path are required")
	}
	if _, err := os.Stat(s.path); errors.Is(err, os.ErrNotExist) {
		return []json.RawMessage{}, nil
	} else if err != nil {
		return nil, domain.NewFailure(domain.FailureUnavailable, "run.state.load", "run-state journal is unavailable")
	}
	events, err := readAndRepairTail[state.Event](s.path)
	if err != nil {
		return nil, err
	}
	history := make([]json.RawMessage, 0)
	for _, event := range events {
		if event.RunID != runID {
			continue
		}
		raw, err := json.Marshal(event)
		if err != nil {
			return nil, domain.NewFailure(domain.FailureInternal, "run.state.load", "run-state event cannot be encoded")
		}
		history = append(history, raw)
	}
	return history, nil
}

func acquireJournalLock(path string) (*lockFile, error) {
	lockPath := path + ".LOCK"
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, domain.NewFailure(domain.FailureConflict, "run.state.ownership", "another writer owns the run-state journal")
		}
		return nil, domain.NewFailure(domain.FailureUnavailable, "run.state.ownership", "run-state journal ownership is unavailable")
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(lockPath)
		return nil, domain.NewFailure(domain.FailureUnavailable, "run.state.ownership", "run-state journal ownership could not be synced")
	}
	return &lockFile{path: lockPath, file: file}, nil
}

func validateNextStateEvent(history []state.Event, next state.Event) error {
	occurredAt, err := time.Parse(time.RFC3339Nano, next.OccurredAt)
	if err != nil || occurredAt.IsZero() {
		return domain.NewFailure(domain.FailureInvalidInput, "run.state.append", "run-state event time is invalid")
	}
	if next.Status == state.Running {
		expected, started, err := state.StartAttempt(history, next.RunID, next.PlanID, next.OwnerToken, occurredAt)
		if err != nil {
			return err
		}
		if !started || expected != next {
			return domain.NewFailure(domain.FailureConflict, "run.state.append", "run attempt is stale, duplicate, or inconsistent with history")
		}
		return nil
	}
	if len(history) == 0 {
		return domain.NewFailure(domain.FailureConflict, "run.state.append", "terminal transition has no running attempt")
	}
	if (next.Status == state.Failed || next.Status == state.Cancelled) &&
		(next.FailureCategory == "" || next.FailureStage == "") {
		return domain.NewFailure(domain.FailureInvalidInput, "run.state.append", "terminal failure requires a stable category and stage")
	}
	current := history[len(history)-1]
	var expected state.Event
	if next.Status == state.Failed || next.Status == state.Cancelled {
		expected, err = state.Transition(current, next.OwnerToken, next.Status, next.FailureCategory, occurredAt, next.FailureStage)
	} else {
		expected, err = state.Transition(current, next.OwnerToken, next.Status, next.FailureCategory, occurredAt)
	}
	if err != nil {
		return err
	}
	if expected != next || expected.Attempt != current.Attempt || expected.PlanID != current.PlanID {
		return domain.NewFailure(domain.FailureConflict, "run.state.append", "terminal transition does not match the current fenced attempt")
	}
	return nil
}

var _ ports.RunStateStore = fileRunStateStore{}
