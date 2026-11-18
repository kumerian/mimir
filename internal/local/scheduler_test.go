package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/scheduler"
)

func TestEvaluateSchedulesPersistsStableRequestWithoutStartingRun(t *testing.T) {
	root, _, stateDir, candidate := setupLocalRun(t, "schedule-1", `{"id":1}`)
	document := strings.Replace(
		draftMetadata("schedule-1"),
		`"path":"orders.jsonl"`,
		`"path":"orders.jsonl","schedule":{"enabled":true,"intervalSeconds":60,"startsAt":"2026-11-18T10:00:00Z","retryLimit":2,"concurrencyLimit":1}`,
		1,
	)
	if err := os.WriteFile(candidate, []byte(document), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
		t.Fatal(err)
	}
	options := ScheduleOptions{
		StateDir: stateDir, Actor: "local-user", AllowLocal: true,
		At: time.Date(2026, 11, 18, 10, 4, 17, 0, time.UTC),
	}
	first, err := EvaluateSchedules(context.Background(), options)
	if err != nil || len(first.Requests) != 1 || !first.Decisions[0].Eligible {
		t.Fatalf("first evaluation=%+v err=%v", first, err)
	}
	options.At = options.At.Add(20 * time.Second)
	second, err := EvaluateSchedules(context.Background(), options)
	if err != nil || len(second.Requests) != 1 ||
		second.Requests[0].RequestID != first.Requests[0].RequestID {
		t.Fatalf("duplicate interval evaluation=%+v err=%v", second, err)
	}
	requests, err := readAndRepairTail[persistedRequest](filepath.Join(stateDir, "state", "schedule-requests.jsonl"))
	if err != nil || len(requests) != 1 {
		t.Fatalf("persisted requests=%+v err=%v", requests, err)
	}
	if _, err := os.Stat(filepath.Join(root, "state", "data", "destination.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("schedule evaluation started ingestion: %v", err)
	}
}

func TestEvaluateSchedulesStopsOnCancellationWithoutCreatingRequests(t *testing.T) {
	_, _, stateDir, candidate := setupLocalRun(t, "schedule-cancel", `{"id":1}`)
	document := strings.Replace(
		draftMetadata("schedule-cancel"),
		`"path":"orders.jsonl"`,
		`"path":"orders.jsonl","schedule":{"enabled":true,"intervalSeconds":60}`,
		1,
	)
	if err := os.WriteFile(candidate, []byte(document), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := EvaluateSchedules(ctx, ScheduleOptions{
		StateDir: stateDir, Actor: "local-user", AllowLocal: true,
		At: time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC),
	})
	var failure domain.Failure
	if !errors.As(err, &failure) || failure.Category != domain.FailureCancelled {
		t.Fatalf("cancelled schedule evaluation=%#v, want cancelled failure", err)
	}
	if _, err := os.Stat(filepath.Join(stateDir, "state", "schedule-requests.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("cancelled evaluation created a request journal: %v", err)
	}
}

func TestStopPendingScheduleRequestsAcknowledgesWithoutChangingRunState(t *testing.T) {
	_, _, stateDir, candidate := setupLocalRun(t, "schedule-stop", `{"id":1}`)
	document := strings.Replace(
		draftMetadata("schedule-stop"),
		`"path":"orders.jsonl"`,
		`"path":"orders.jsonl","schedule":{"enabled":true,"intervalSeconds":60}`,
		1,
	)
	if err := os.WriteFile(candidate, []byte(document), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
		t.Fatal(err)
	}
	_, err := EvaluateSchedules(context.Background(), ScheduleOptions{
		StateDir: stateDir, Actor: "local-user", AllowLocal: true,
		At: time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := StopPendingScheduleRequests(context.Background(), stateDir, "local-user", true, "shutdown"); err != nil {
		t.Fatal(err)
	}
	acks, err := readAndRepairTail[scheduler.Acknowledgement](filepath.Join(stateDir, "state", "schedule-request-acks.jsonl"))
	if err != nil || len(acks) != 1 || acks[0].State != "stopped" || acks[0].Reason != "shutdown" {
		t.Fatalf("request acknowledgements=%+v err=%v", acks, err)
	}
	if info, err := os.Stat(filepath.Join(stateDir, "state", "runs.jsonl")); err != nil || info.Size() != 0 {
		t.Fatalf("stopping pending request changed run execution state: info=%v err=%v", info, err)
	}
	if err := StopPendingScheduleRequests(context.Background(), stateDir, "local-user", true, "shutdown"); err != nil {
		t.Fatal(err)
	}
	acks, err = readAndRepairTail[scheduler.Acknowledgement](filepath.Join(stateDir, "state", "schedule-request-acks.jsonl"))
	if err != nil || len(acks) != 1 {
		t.Fatalf("repeated stop was not idempotent: acks=%+v err=%v", acks, err)
	}
	stoppedEvaluation, err := EvaluateSchedules(context.Background(), ScheduleOptions{
		StateDir: stateDir, Actor: "local-user", AllowLocal: true,
		At: time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC),
	})
	if err != nil || len(stoppedEvaluation.Requests) != 0 {
		t.Fatalf("stopped request was returned as pending again: result=%+v err=%v", stoppedEvaluation, err)
	}
}

func TestScheduleCancellationAcknowledgesRequestsPersistedEarlierInTheEvaluation(t *testing.T) {
	_, _, stateDir, candidate := setupLocalRun(t, "schedule-mid-cancel", `{"id":1}`)
	document := strings.Replace(
		draftMetadata("schedule-mid-cancel"),
		`"objects":[{"id":"orders","path":"orders.jsonl"}]`,
		`"objects":[{"id":"orders","path":"orders.jsonl","schedule":{"enabled":true,"intervalSeconds":60}},{"id":"customers","path":"customers.jsonl","schedule":{"enabled":true,"intervalSeconds":60}}]`,
		1,
	)
	if err := os.WriteFile(candidate, []byte(document), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	_, err := evaluateSchedules(ctx, ScheduleOptions{
		StateDir: stateDir, Actor: "local-user", AllowLocal: true,
		At: time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC),
	}, cancel)
	cancel()
	var failure domain.Failure
	if !errors.As(err, &failure) || failure.Category != domain.FailureCancelled {
		t.Fatalf("mid-evaluation cancellation=%#v, want cancelled failure", err)
	}
	requests, err := readAndRepairTail[persistedRequest](filepath.Join(stateDir, "state", "schedule-requests.jsonl"))
	if err != nil || len(requests) != 1 {
		t.Fatalf("persisted requests=%+v err=%v", requests, err)
	}
	acks, err := readAndRepairTail[scheduler.Acknowledgement](filepath.Join(stateDir, "state", "schedule-request-acks.jsonl"))
	if err != nil || len(acks) != 1 || acks[0].RequestID != requests[0].Request.RequestID ||
		acks[0].Reason != "cancelled" {
		t.Fatalf("cancellation acknowledgements=%+v err=%v", acks, err)
	}
	if info, err := os.Stat(filepath.Join(stateDir, "state", "runs.jsonl")); err != nil || info.Size() != 0 {
		t.Fatalf("request cancellation changed run execution state: info=%v err=%v", info, err)
	}
}
