package local

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/ports"
	"github.com/kumerian/mimir/internal/state"
)

func TestFileRunStateStoreFencesStaleOwnersAndPreservesRetryHistory(t *testing.T) {
	store := fileRunStateStore{path: t.TempDir() + "/state/runs.jsonl"}
	ctx := context.Background()
	now := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)
	first, created, err := state.StartAttempt(nil, "run-1", "plan-1", "owner-1", now)
	if err != nil || !created {
		t.Fatalf("StartAttempt() created=%v err=%v", created, err)
	}
	appendStateEvent(t, store, ctx, first)

	stale, err := state.Transition(first, "owner-stale", state.Succeeded, "", now.Add(time.Second))
	if err == nil {
		t.Fatal("state policy accepted stale owner before store verification")
	}
	stale = first
	stale.OwnerToken = "owner-stale"
	stale.Status = state.Succeeded
	stale.OccurredAt = now.Add(time.Second).Format(time.RFC3339Nano)
	if err := appendStateEventError(store, ctx, stale); err == nil {
		t.Fatal("run-state store accepted stale owner transition")
	}

	failed, err := state.Transition(first, "owner-1", state.Failed, domain.FailureUnavailable, now.Add(2*time.Second), "run.processing")
	if err != nil {
		t.Fatal(err)
	}
	appendStateEvent(t, store, ctx, failed)
	second, created, err := state.StartAttempt([]state.Event{first, failed}, "run-1", "plan-1", "owner-2", now.Add(3*time.Second))
	if err != nil || !created || second.Attempt != 2 {
		t.Fatalf("retry=%+v created=%v err=%v", second, created, err)
	}
	appendStateEvent(t, store, ctx, second)
	stale = first
	stale.Status = state.Succeeded
	stale.OccurredAt = now.Add(4 * time.Second).Format(time.RFC3339Nano)
	if err := appendStateEventError(store, ctx, stale); err == nil {
		t.Fatal("older attempt advanced after newer ownership was acquired")
	}
	succeeded, err := state.Transition(second, "owner-2", state.Succeeded, "", now.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	appendStateEvent(t, store, ctx, succeeded)

	history, err := store.LoadHistory(ctx, "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 4 {
		t.Fatalf("run history length=%d, want four immutable lifecycle events", len(history))
	}
	var persistedFailure state.Event
	if err := json.Unmarshal(history[1], &persistedFailure); err != nil {
		t.Fatal(err)
	}
	if persistedFailure.Status != state.Failed || persistedFailure.FailureCategory != domain.FailureUnavailable ||
		persistedFailure.FailureStage != "run.processing" {
		t.Fatalf("prior failure evidence was overwritten: %+v", persistedFailure)
	}
	if got := store.Capabilities().ContractVersion; got != ports.RunStateStoreVersion {
		t.Fatalf("run-state contract version=%q", got)
	}
}

func TestFileRunStateStoreAllowsOnlyOneConcurrentInitialOwner(t *testing.T) {
	store := fileRunStateStore{path: t.TempDir() + "/state/runs.jsonl"}
	now := time.Now().UTC()
	events := []state.Event{
		{ContractVersion: state.ContractVersion, RunID: "race", PlanID: "plan", Attempt: 1, Status: state.Running, OwnerToken: "owner-a", OccurredAt: now.Format(time.RFC3339Nano)},
		{ContractVersion: state.ContractVersion, RunID: "race", PlanID: "plan", Attempt: 1, Status: state.Running, OwnerToken: "owner-b", OccurredAt: now.Format(time.RFC3339Nano)},
	}
	var wait sync.WaitGroup
	errors := make(chan error, len(events))
	wait.Add(len(events))
	for _, event := range events {
		go func(event state.Event) {
			defer wait.Done()
			errors <- appendStateEventError(store, context.Background(), event)
		}(event)
	}
	wait.Wait()
	close(errors)
	successes := 0
	for err := range errors {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent initial owners accepted=%d, want exactly one", successes)
	}
	history, err := store.LoadHistory(context.Background(), "race")
	if err != nil || len(history) != 1 {
		t.Fatalf("persisted race history=%d err=%v", len(history), err)
	}
}

func TestFileRunStateStoreRejectsUnsupportedVersionWithoutSideEffects(t *testing.T) {
	store := fileRunStateStore{path: t.TempDir() + "/state/runs.jsonl"}
	event := state.Event{
		ContractVersion: "mimir.run-state/v99", RunID: "run", PlanID: "plan",
		Attempt: 1, Status: state.Running, OwnerToken: "owner",
		OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := appendStateEventError(store, context.Background(), event); err == nil {
		t.Fatal("run-state store accepted unsupported version")
	}
	history, err := store.LoadHistory(context.Background(), "run")
	if err != nil || len(history) != 0 {
		t.Fatalf("unsupported version caused partial state: history=%d err=%v", len(history), err)
	}
}

func appendStateEvent(t *testing.T, store fileRunStateStore, ctx context.Context, event state.Event) {
	t.Helper()
	if err := appendStateEventError(store, ctx, event); err != nil {
		t.Fatalf("AppendTransition(%s/%d/%s): %v", event.RunID, event.Attempt, event.Status, err)
	}
}

func appendStateEventError(store fileRunStateStore, ctx context.Context, event state.Event) error {
	raw, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return store.AppendTransition(ctx, event.RunID, raw)
}
