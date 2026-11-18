package local

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kumerian/mimir/internal/ports"
)

func TestFileAuditStoreDeduplicatesRequiredEvidence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit", "events.jsonl")
	store := fileAuditStore{path: path}
	event := auditEvent{
		EventID: digest([]byte("event-1")), Operation: "run.committed", Actor: "local-user",
		MetadataRevision: "revision-1", PlanID: "plan-1", RunID: "run-1", Attempt: 2,
		OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := store.Append(context.Background(), raw); err != nil {
			t.Fatal(err)
		}
	}
	events, err := readAndRepairTail[auditEvent](path)
	if err != nil || len(events) != 1 {
		t.Fatalf("audit evidence count=%d err=%v", len(events), err)
	}
	if events[0].Attempt != 2 || events[0].RunID != event.RunID || events[0].PlanID != event.PlanID {
		t.Fatalf("audit correlation was not retained: %+v", events[0])
	}
	if store.Capabilities().ContractVersion != ports.AuditStoreVersion {
		t.Fatal("audit store did not declare its port version")
	}
	unsafe := []byte(`{"eventId":"` + digest([]byte("unsafe")) + `","operation":"synthetic-canary","actor":"local-user","metadataRevision":"revision-1","runId":"run-1","occurredAt":"2026-11-18T10:00:00Z"}`)
	if err := store.Append(context.Background(), unsafe); err == nil {
		t.Fatal("audit adapter accepted secret-shaped unbounded operation text")
	}
	unsafeActor := []byte(`{"eventId":"` + digest([]byte("unsafe-actor")) + `","operation":"run.committed","actor":"synthetic-canary","metadataRevision":"revision-1","runId":"run-1","occurredAt":"2026-11-18T10:00:00Z"}`)
	if err := store.Append(context.Background(), unsafeActor); err == nil {
		t.Fatal("audit adapter accepted secret-shaped actor text")
	}
}

func TestFileTelemetryStoreCorrelatesAndRejectsSensitiveExtensions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "telemetry", "events.jsonl")
	store := fileTelemetry{path: path}
	event := telemetryEvent{
		EventID: digest([]byte("event-1")), Name: "run.succeeded", Stage: "completion", Outcome: "succeeded",
		MetadataRevision: "revision-1", PlanID: "plan-1", RunID: "run-1", Attempt: 2,
		OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Emit(context.Background(), raw); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(contents), "secret") || strings.Contains(string(contents), "canary") {
		t.Fatalf("telemetry serialized sensitive content: %s", contents)
	}
	var persisted telemetryEvent
	if err := json.Unmarshal(contents, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.RunID != event.RunID || persisted.PlanID != event.PlanID ||
		persisted.MetadataRevision != event.MetadataRevision || persisted.Attempt != event.Attempt {
		t.Fatalf("telemetry correlation mismatch: %+v", persisted)
	}
	unsafe := []byte(`{"eventId":"unsafe","name":"run.succeeded","stage":"completion","outcome":"succeeded","metadataRevision":"revision-1","planId":"plan-1","runId":"run-1","attempt":2,"occurredAt":"2026-11-18T10:00:00Z","secret":"synthetic-canary"}`)
	if err := store.Emit(context.Background(), unsafe); err == nil {
		t.Fatal("telemetry adapter accepted an unmodeled sensitive field")
	}
	if store.Capabilities().ContractVersion != ports.TelemetryVersion {
		t.Fatal("telemetry adapter did not declare its port version")
	}
	unsafeEvent := event
	unsafeEvent.Name = "synthetic-canary"
	unsafeRaw, _ := json.Marshal(unsafeEvent)
	if err := store.Emit(context.Background(), unsafeRaw); err == nil {
		t.Fatal("telemetry accepted secret-shaped event names")
	}
	unsafeEvent = event
	unsafeEvent.FailureCategory = "synthetic-canary"
	unsafeRaw, _ = json.Marshal(unsafeEvent)
	if err := store.Emit(context.Background(), unsafeRaw); err == nil {
		t.Fatal("telemetry accepted secret-shaped failure categories")
	}
}

func TestFileTelemetryCancellationHasNoSideEffect(t *testing.T) {
	path := filepath.Join(t.TempDir(), "telemetry", "events.jsonl")
	store := fileTelemetry{path: path}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	raw, _ := json.Marshal(telemetryEvent{
		EventID: digest([]byte("event-1")), Name: "run.started", Stage: "execution", Outcome: "started",
		MetadataRevision: "revision-1", PlanID: "plan-1", RunID: "run-1", Attempt: 1,
		OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err := store.Emit(ctx, raw); err == nil {
		t.Fatal("telemetry adapter ignored cancellation")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("cancelled telemetry created an output: %v", err)
	}
}

func TestRunCorrelatesDurableAuditAndBestEffortTelemetry(t *testing.T) {
	_, sourceDir, stateDir, candidate := setupLocalRun(t, "observed-run", `{"id":12}`)
	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
		t.Fatal(err)
	}
	result, err := Run(context.Background(), localRunOptions(sourceDir, stateDir))
	if err != nil {
		t.Fatal(err)
	}
	audits, err := readAndRepairTail[auditEvent](filepath.Join(stateDir, "audit", "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	telemetry, err := readAndRepairTail[telemetryEvent](filepath.Join(stateDir, "telemetry", "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var committed auditEvent
	for _, event := range audits {
		if event.Operation == "run.committed" {
			committed = event
		}
	}
	var succeeded telemetryEvent
	for _, event := range telemetry {
		if event.Name == "run.succeeded" {
			succeeded = event
		}
	}
	if committed.RunID == "" || committed.RunID != result.RunID ||
		succeeded.RunID != committed.RunID || succeeded.PlanID != committed.PlanID ||
		succeeded.MetadataRevision != committed.MetadataRevision || succeeded.Attempt != committed.Attempt {
		t.Fatalf("audit and telemetry correlation differ: audit=%+v telemetry=%+v result=%+v", committed, succeeded, result)
	}
}
