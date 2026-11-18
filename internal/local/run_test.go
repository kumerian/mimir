package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/state"
)

func TestRunCommitsOnceAndResumesFromSnapshot(t *testing.T) {
	root, sourceDir, stateDir, candidate := setupLocalRun(t, "run-1", `{"id":1,"name":"synthetic"}`)
	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	options := localRunOptions(sourceDir, stateDir)
	first, err := Run(context.Background(), options)
	if err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
	second, err := Run(context.Background(), options)
	if err != nil {
		t.Fatalf("second Run() error = %v", err)
	}
	if first.RunID != second.RunID || !second.AlreadyCommitted {
		t.Fatalf("duplicate run was not reconciled: first=%+v second=%+v", first, second)
	}
	committed, err := ReadCommitted(filepath.Join(root, "state", "data", "destination.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(committed) != 1 || committed[0].BatchID != first.RunID {
		t.Fatalf("unexpected committed batches: %+v", committed)
	}
	states, err := readAndRepairTail[state.Event](filepath.Join(stateDir, "state", "runs.jsonl"))
	if err != nil || len(states) != 2 || states[0].Status != state.Running || states[1].Status != state.Succeeded {
		t.Fatalf("run lifecycle evidence=%+v err=%v", states, err)
	}
}

func TestRunAppliesPinnedTransformAndPersistsQualityEvidence(t *testing.T) {
	root, sourceDir, stateDir, candidate := setupLocalRun(t, "rules-1", `{"id":1,"name":" Example "}`)
	document := strings.Replace(
		draftMetadata("rules-1"),
		`"path":"orders.jsonl"`,
		`"path":"orders.jsonl","transformations":[{"id":"trim-name","version":"v1","operation":"trim","field":"name"}],"qualityRules":[{"id":"name-required","version":"v1","operation":"required","field":"name"}],"qualityFailureDisposition":"reject"`,
		1,
	)
	if err := os.WriteFile(candidate, []byte(document), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	result, err := Run(context.Background(), localRunOptions(sourceDir, stateDir))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	batches, err := ReadCommitted(filepath.Join(root, "state", "data", "destination.jsonl"))
	if err != nil || len(batches) != 1 {
		t.Fatalf("ReadCommitted() batches=%d err=%v", len(batches), err)
	}
	if got := string(batches[0].Records[0]); got != `{"id":1,"name":"Example"}` {
		t.Fatalf("published transformed record = %s", got)
	}
	journal, err := os.ReadFile(filepath.Join(root, "state", "data", "destination.jsonl"))
	if err != nil || !strings.Contains(string(journal), `"originalRecords":[{"id":1,"name":" Example "}]`) ||
		!strings.Contains(string(journal), `"qualityResults":[`) {
		t.Fatalf("original data and quality evidence were not persisted: err=%v journal=%s", err, journal)
	}
	planBytes, err := os.ReadFile(filepath.Join(stateDir, "plans", result.RunID+".json"))
	if err != nil || !strings.Contains(string(planBytes), `"transformations":["trim-name@v1"]`) ||
		!strings.Contains(string(planBytes), `"qualityRules":["name-required@v1"]`) {
		t.Fatalf("run plan did not pin rule versions: err=%v plan=%s", err, planBytes)
	}
}

func TestRunQuarantinesQualityFailuresBeforeCheckpoint(t *testing.T) {
	root, sourceDir, stateDir, candidate := setupLocalRun(t, "quality-quarantine-1", `{"id":2}`)
	document := strings.Replace(
		draftMetadata("quality-quarantine-1"),
		`"path":"orders.jsonl"`,
		`"path":"orders.jsonl","qualityRules":[{"id":"name-required","version":"v1","operation":"required","field":"name"}],"qualityFailureDisposition":"quarantine"`,
		1,
	)
	if err := os.WriteFile(candidate, []byte(document), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(context.Background(), localRunOptions(sourceDir, stateDir)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	destination, err := os.ReadFile(filepath.Join(root, "state", "data", "destination.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(destination), `"records":[]`) {
		t.Fatalf("quality-failed record was published: %s", destination)
	}
	quarantine, err := os.ReadFile(filepath.Join(root, "state", "data", "quarantine.jsonl"))
	if err != nil || !strings.Contains(string(quarantine), `"originalRecord":{"id":2}`) ||
		!strings.Contains(string(quarantine), `"disposition":"quarantine"`) {
		t.Fatalf("quality-failed record was not durably quarantined: err=%v journal=%s", err, quarantine)
	}
}

func TestRunReconcilesQuarantineBeforeRecreatingCheckpoint(t *testing.T) {
	root, sourceDir, stateDir, candidate := setupLocalRun(t, "quality-quarantine-recovery-1", `{"id":3}`)
	document := strings.Replace(
		draftMetadata("quality-quarantine-recovery-1"),
		`"path":"orders.jsonl"`,
		`"path":"orders.jsonl","qualityRules":[{"id":"name-required","version":"v1","operation":"required","field":"name"}],"qualityFailureDisposition":"quarantine"`,
		1,
	)
	if err := os.WriteFile(candidate, []byte(document), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
		t.Fatal(err)
	}
	options := localRunOptions(sourceDir, stateDir)
	first, err := Run(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(stateDir, "data", "quarantine.jsonl")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(stateDir, "state", "checkpoints.jsonl")); err != nil {
		t.Fatal(err)
	}
	recovered, err := Run(context.Background(), options)
	if err != nil || recovered.RunID != first.RunID {
		t.Fatalf("quarantine recovery result=%+v err=%v", recovered, err)
	}
	destination, err := ReadCommitted(filepath.Join(root, "state", "data", "destination.jsonl"))
	if err != nil || len(destination) != 1 || len(destination[0].Records) != 0 {
		t.Fatalf("destination commit duplicated or changed: batches=%+v err=%v", destination, err)
	}
	quarantine, err := os.ReadFile(filepath.Join(stateDir, "data", "quarantine.jsonl"))
	if err != nil || strings.Count(string(quarantine), `"batchId"`) != 1 {
		t.Fatalf("quarantine recovery duplicated or omitted evidence: err=%v journal=%s", err, quarantine)
	}
}

func TestRunRejectsQualityFailureAndAuditsWithoutPublishing(t *testing.T) {
	root, sourceDir, stateDir, candidate := setupLocalRun(t, "quality-reject-1", `{"id":4}`)
	document := strings.Replace(
		draftMetadata("quality-reject-1"),
		`"path":"orders.jsonl"`,
		`"path":"orders.jsonl","qualityRules":[{"id":"name-required","version":"v1","operation":"required","field":"name"}],"qualityFailureDisposition":"reject"`,
		1,
	)
	if err := os.WriteFile(candidate, []byte(document), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
		t.Fatal(err)
	}
	_, err := Run(context.Background(), localRunOptions(sourceDir, stateDir))
	failure, ok := err.(domain.Failure)
	if !ok || failure.Category != domain.FailurePolicyRejected {
		t.Fatalf("Run() failure = %#v, want policy_rejected", err)
	}
	if _, err := os.Stat(filepath.Join(root, "state", "data", "destination.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("quality reject created destination data: %v", err)
	}
	audits, err := readAndRepairTail[auditEvent](filepath.Join(stateDir, "audit", "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, event := range audits {
		if event.Operation == "run.failed" && event.FailureCategory == string(domain.FailurePolicyRejected) {
			found = true
		}
	}
	if !found {
		t.Fatal("quality rejection was not durably audited")
	}
	states, err := readAndRepairTail[state.Event](filepath.Join(stateDir, "state", "runs.jsonl"))
	if err != nil || len(states) != 2 || states[1].Status != state.Failed ||
		states[1].FailureCategory != domain.FailurePolicyRejected {
		t.Fatalf("quality rejection lifecycle=%+v err=%v", states, err)
	}
}

func TestReplayReconcilesOriginalPinnedRunWithoutDuplicatingDestination(t *testing.T) {
	root, sourceDir, stateDir, candidate := setupLocalRun(t, "replay-1", `{"id":5}`)
	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
		t.Fatal(err)
	}
	options := localRunOptions(sourceDir, stateDir)
	original, err := Run(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := Replay(context.Background(), options, original.RunID, "operator_request")
	if err != nil {
		t.Fatalf("Replay() error = %v", err)
	}
	if replayed.OriginalRunID != original.RunID || replayed.Result.RunID != original.RunID ||
		!replayed.Result.AlreadyCommitted || replayed.ReplayPlanID == "" {
		t.Fatalf("unexpected replay result: %+v", replayed)
	}
	batches, err := ReadCommitted(filepath.Join(root, "state", "data", "destination.jsonl"))
	if err != nil || len(batches) != 1 {
		t.Fatalf("replay duplicated destination: batches=%d err=%v", len(batches), err)
	}
	audit, err := os.ReadFile(filepath.Join(stateDir, "audit", "events.jsonl"))
	if err != nil || !strings.Contains(string(audit), `"operation":"replay.prepared"`) {
		t.Fatalf("replay preparation was not audited: err=%v audit=%s", err, audit)
	}
}

func TestRunRejectsWithoutActivatedMetadataBeforeOpeningDestination(t *testing.T) {
	_, sourceDir, stateDir, _ := setupLocalRun(t, "draft-1", `{"id":1}`)
	_, err := Run(context.Background(), localRunOptions(sourceDir, stateDir))
	if err == nil {
		t.Fatal("Run() accepted metadata without an activation record")
	}
	if _, statErr := os.Stat(stateDir); !os.IsNotExist(statErr) {
		t.Fatalf("planning rejection created state directory: stat err=%v", statErr)
	}
}

func TestRunFailsClosedWithoutExplicitLocalAuthorization(t *testing.T) {
	_, sourceDir, stateDir, candidate := setupLocalRun(t, "auth-1", `{"id":1}`)
	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
		t.Fatal(err)
	}
	options := localRunOptions(sourceDir, stateDir)
	options.AllowLocal = false
	if _, err := Run(context.Background(), options); err == nil {
		t.Fatal("Run() succeeded without explicit local authorization")
	}
	audits, err := readAndRepairTail[auditEvent](filepath.Join(stateDir, "audit", "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	foundDenial := false
	for _, event := range audits {
		if event.Operation == "access.denied" && event.FailureCategory == "access_denied" {
			foundDenial = true
		}
	}
	if !foundDenial {
		t.Fatal("denied run did not produce redacted audit evidence")
	}
}

func TestActivateCreatesImmutableAuthorizedActiveRevision(t *testing.T) {
	root := t.TempDir()
	candidate := filepath.Join(root, "candidate.json")
	if err := os.WriteFile(candidate, []byte(draftMetadata("rev-a")), 0o600); err != nil {
		t.Fatal(err)
	}
	stateDir := filepath.Join(root, "state")
	result, err := Activate(context.Background(), candidate, stateDir, "local-user", true)
	if err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	if result.Revision != "rev-a" || result.Status != "active" {
		t.Fatalf("unexpected activation result: %+v", result)
	}
	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
		t.Fatalf("idempotent Activate() error = %v", err)
	}
	if _, err := Activate(context.Background(), candidate, stateDir, "other-actor", true); err == nil {
		t.Fatal("Activate() accepted an unrecognized actor")
	}
}

func TestAuthorPersistsDraftAndListShowsImmutableHistory(t *testing.T) {
	root := t.TempDir()
	candidate := filepath.Join(root, "candidate.json")
	stateDir := filepath.Join(root, "state")
	if err := os.WriteFile(candidate, []byte(draftMetadata("authored-1")), 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := Author(context.Background(), candidate, stateDir, "local-user", true)
	if err != nil {
		t.Fatalf("Author() error = %v", err)
	}
	second, err := Author(context.Background(), candidate, stateDir, "local-user", true)
	if err != nil || second.Digest != first.Digest {
		t.Fatalf("idempotent Author() result=%+v err=%v", second, err)
	}
	revisions, err := ListMetadata(stateDir, "local-user", true)
	if err != nil || len(revisions) != 1 || revisions[0].Status != "draft" {
		t.Fatalf("ListMetadata() revisions=%+v err=%v", revisions, err)
	}
	audits, err := readAndRepairTail[auditEvent](filepath.Join(stateDir, "audit", "events.jsonl"))
	if err != nil || len(audits) != 1 || audits[0].Operation != "metadata.authored" {
		t.Fatalf("authoring audit=%+v err=%v", audits, err)
	}
}

func TestActivateRequiresExpectedRevisionWhenSuperseding(t *testing.T) {
	root := t.TempDir()
	stateDir := filepath.Join(root, "state")
	firstCandidate := filepath.Join(root, "first.json")
	secondCandidate := filepath.Join(root, "second.json")
	if err := os.WriteFile(firstCandidate, []byte(draftMetadata("activation-first")), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secondCandidate, []byte(draftMetadata("activation-second")), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Activate(context.Background(), firstCandidate, stateDir, "local-user", true); err != nil {
		t.Fatal(err)
	}
	if _, err := Activate(context.Background(), secondCandidate, stateDir, "local-user", true); err == nil {
		t.Fatal("Activate() superseded an active revision without an expected-current precondition")
	}
	if _, err := Activate(context.Background(), secondCandidate, stateDir, "local-user", true, "stale-revision"); err == nil {
		t.Fatal("Activate() accepted a stale expected-current revision")
	}
	if _, err := Activate(context.Background(), secondCandidate, stateDir, "local-user", true, "activation-first"); err != nil {
		t.Fatalf("Activate() with the current expected revision: %v", err)
	}
}

func TestConcurrentActivationsWithSameExpectedRevisionSelectAtMostOne(t *testing.T) {
	root := t.TempDir()
	stateDir := filepath.Join(root, "state")
	paths := []string{filepath.Join(root, "a.json"), filepath.Join(root, "b.json")}
	for i, revision := range []string{"race-a", "race-b"} {
		if err := os.WriteFile(paths[i], []byte(draftMetadata(revision)), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var wait sync.WaitGroup
	errors := make(chan error, len(paths))
	wait.Add(len(paths))
	for _, path := range paths {
		go func(candidate string) {
			defer wait.Done()
			_, err := Activate(context.Background(), candidate, stateDir, "local-user", true, "")
			errors <- err
		}(path)
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
		t.Fatalf("concurrent activation successes = %d, want exactly one", successes)
	}
	pointers, err := readAndRepairTail[activationPointer](filepath.Join(stateDir, "metadata", "active.jsonl"))
	if err != nil || len(pointers) != 1 {
		t.Fatalf("active history=%+v err=%v", pointers, err)
	}
}

func TestRunReconcilesCommittedBatchWhenCheckpointAndRunAuditAreMissing(t *testing.T) {
	root, sourceDir, stateDir, candidate := setupLocalRun(t, "recover-1", `{"id":7}`)
	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
		t.Fatal(err)
	}
	options := localRunOptions(sourceDir, stateDir)
	first, err := Run(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(stateDir, "state", "checkpoints.jsonl")); err != nil {
		t.Fatal(err)
	}
	auditPath := filepath.Join(stateDir, "audit", "events.jsonl")
	auditData, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatal(err)
	}
	var activationAudit strings.Builder
	for _, line := range strings.Split(string(auditData), "\n") {
		if strings.Contains(line, `"operation":"metadata.activated"`) {
			activationAudit.WriteString(line)
			activationAudit.WriteByte('\n')
		}
	}
	if err := os.WriteFile(auditPath, []byte(activationAudit.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := Run(context.Background(), options)
	if err != nil {
		t.Fatalf("reconcile Run() error = %v", err)
	}
	if second.RunID != first.RunID || !second.AlreadyCommitted {
		t.Fatalf("expected destination reconciliation: first=%+v second=%+v", first, second)
	}
	batches, err := ReadCommitted(filepath.Join(root, "state", "data", "destination.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(batches) != 1 {
		t.Fatalf("reconciliation duplicated committed data: %d batches", len(batches))
	}
}

func TestRunRecoversAfterInjectedDurableBoundaryFailures(t *testing.T) {
	stages := []string{
		"before_plan_audit",
		"after_plan_audit",
		"before_state_start",
		"after_state_started",
		"before_destination_commit",
		"after_destination_commit",
		"before_quarantine_commit",
		"after_quarantine_commit",
		"before_checkpoint_commit",
		"after_checkpoint_commit",
		"before_run_audit",
		"after_run_audit",
		"before_terminal_state",
		"after_terminal_state",
	}
	for _, stage := range stages {
		t.Run(stage, func(t *testing.T) {
			root, sourceDir, stateDir, candidate := setupLocalRun(t, "fault-"+stage, `{"id":11}`)
			revision := strings.TrimPrefix("fault-"+stage, "fault-")
			quarantineBoundary := strings.Contains(stage, "quarantine")
			if quarantineBoundary {
				if err := os.WriteFile(candidate, []byte(strings.Replace(
					draftMetadata(revision),
					`"path":"orders.jsonl"`,
					`"path":"orders.jsonl","qualityRules":[{"id":"name-required","version":"v1","operation":"required","field":"name"}],"qualityFailureDisposition":"quarantine"`,
					1,
				)), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
				t.Fatal(err)
			}
			options := localRunOptions(sourceDir, stateDir)
			injected := false
			func() {
				defer func() {
					crash, ok := recover().(injectedRunCrash)
					if !ok || crash.stage != stage {
						t.Fatalf("run did not stop at injected boundary %q: recovered=%v", stage, crash)
					}
				}()
				_, _ = runWithFaultHook(context.Background(), options, func(current string) bool {
					if current == stage {
						injected = true
						return true
					}
					return false
				})
			}()
			if !injected {
				t.Fatalf("fault boundary %q was not reached", stage)
			}
			if err := os.Remove(filepath.Join(stateDir, "LOCK")); err != nil {
				t.Fatalf("simulate operator-confirmed process lock cleanup: %v", err)
			}
			result, err := Run(context.Background(), options)
			if err != nil {
				t.Fatalf("recovered Run() error = %v", err)
			}
			if result.RunID == "" {
				t.Fatal("recovered run has no stable identity")
			}
			batches, err := ReadCommitted(filepath.Join(root, "state", "data", "destination.jsonl"))
			if err != nil || len(batches) != 1 {
				t.Fatalf("recovery committed batches=%d err=%v", len(batches), err)
			}
			if quarantineBoundary {
				entries, err := readAndRepairTail[quarantineBatch](filepath.Join(stateDir, "data", "quarantine.jsonl"))
				if err != nil || len(entries) != 1 {
					t.Fatalf("recovery quarantine entries=%d err=%v", len(entries), err)
				}
			}
			checkpoints, err := readAndRepairTail[checkpoint](filepath.Join(stateDir, "state", "checkpoints.jsonl"))
			if err != nil || len(checkpoints) != 1 {
				t.Fatalf("recovery checkpoints=%d err=%v", len(checkpoints), err)
			}
			audits, err := readAndRepairTail[auditEvent](filepath.Join(stateDir, "audit", "events.jsonl"))
			if err != nil {
				t.Fatal(err)
			}
			committedAudit := 0
			for _, event := range audits {
				if event.Operation == "run.committed" {
					committedAudit++
				}
			}
			wantCommittedAudit := 1
			if stage == "before_terminal_state" || stage == "after_run_audit" {
				wantCommittedAudit = 2 // each durable audit event is retained with its attempt identity
			}
			if committedAudit != wantCommittedAudit {
				t.Fatalf("committed audit count=%d, want %d", committedAudit, wantCommittedAudit)
			}
			states, err := readAndRepairTail[state.Event](filepath.Join(stateDir, "state", "runs.jsonl"))
			if err != nil {
				t.Fatal(err)
			}
			if states[len(states)-1].Status != state.Succeeded {
				t.Fatalf("latest run status=%s, want succeeded", states[len(states)-1].Status)
			}
		})
	}
}

func TestRunRequiredAuditFailureBlocksSuccessAndCanBeReconciled(t *testing.T) {
	root, sourceDir, stateDir, candidate := setupLocalRun(t, "audit-outage", `{"id":13}`)
	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
		t.Fatal(err)
	}
	options := localRunOptions(sourceDir, stateDir)
	_, err := runWithAuditFailure(context.Background(), options, func(eventID string) bool {
		return eventID == "run.committed"
	})
	if err == nil {
		t.Fatal("run reported success when required commit audit append failed")
	}
	batches, readErr := ReadCommitted(filepath.Join(root, "state", "data", "destination.jsonl"))
	if readErr != nil || len(batches) != 1 {
		t.Fatalf("audit outage should preserve one committed batch: batches=%d err=%v", len(batches), readErr)
	}
	audits, err := readAndRepairTail[auditEvent](filepath.Join(stateDir, "audit", "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range audits {
		if event.Operation == "run.committed" {
			t.Fatal("failed audit append produced success-shaped evidence")
		}
	}
	recovered, err := Run(context.Background(), options)
	if err != nil {
		t.Fatalf("run did not recover its required audit outcome: %v", err)
	}
	if recovered.RunID == "" || !recovered.AlreadyCommitted {
		t.Fatalf("recovered result=%+v", recovered)
	}
	batches, err = ReadCommitted(filepath.Join(root, "state", "data", "destination.jsonl"))
	if err != nil || len(batches) != 1 {
		t.Fatalf("audit recovery duplicated destination: batches=%d err=%v", len(batches), err)
	}
}

func TestRunSucceedsWhenBestEffortTelemetrySinkIsUnavailable(t *testing.T) {
	_, sourceDir, stateDir, candidate := setupLocalRun(t, "telemetry-outage", `{"id":14}`)
	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
		t.Fatal(err)
	}
	telemetryPath := filepath.Join(stateDir, "telemetry", "events.jsonl")
	if err := os.MkdirAll(filepath.Dir(telemetryPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(telemetryPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(context.Background(), localRunOptions(sourceDir, stateDir)); err != nil {
		t.Fatalf("best-effort telemetry outage changed run outcome: %v", err)
	}
}

type cancelAfterChecks struct {
	context.Context
	limit int
	mu    sync.Mutex
	calls int
}

func (c *cancelAfterChecks) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	if c.calls >= c.limit {
		return context.Canceled
	}
	return nil
}

func TestRunCancellationDuringProcessingRecordsTerminalStateAndReleasesOwnership(t *testing.T) {
	_, sourceDir, stateDir, candidate := setupLocalRun(t, "cancel-active", `{"id":15}`)
	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
		t.Fatal(err)
	}
	ctx := &cancelAfterChecks{Context: context.Background(), limit: 7}
	_, err := Run(ctx, localRunOptions(sourceDir, stateDir))
	var failure domain.Failure
	if !errors.As(err, &failure) || failure.Category != domain.FailureCancelled {
		t.Fatalf("active cancellation result=%#v, want cancelled failure", err)
	}
	states, err := readAndRepairTail[state.Event](filepath.Join(stateDir, "state", "runs.jsonl"))
	if err != nil || len(states) < 2 || states[len(states)-1].Status != state.Cancelled {
		t.Fatalf("cancelled run history=%+v err=%v", states, err)
	}
	if states[len(states)-1].FailureCategory != domain.FailureCancelled || states[len(states)-1].FailureStage == "" {
		t.Fatalf("cancelled terminal event omitted stable failure details: %+v", states[len(states)-1])
	}
	if _, err := os.Stat(filepath.Join(stateDir, "LOCK")); !os.IsNotExist(err) {
		t.Fatalf("run did not release local ownership after cancellation: %v", err)
	}
	if _, err := os.Stat(filepath.Join(stateDir, "data", "destination.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("cancelled run published destination data: %v", err)
	}
}

func TestRunRepairsOnlyIncompleteFinalJournalLine(t *testing.T) {
	_, sourceDir, stateDir, candidate := setupLocalRun(t, "tail-1", `{"id":8}`)
	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
		t.Fatal(err)
	}
	options := localRunOptions(sourceDir, stateDir)
	if _, err := Run(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(stateDir, "data", "destination.jsonl")
	file, err := os.OpenFile(destination, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("{\"incomplete\":"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(context.Background(), options); err != nil {
		t.Fatalf("Run() did not recover incomplete tail: %v", err)
	}
	batches, err := ReadCommitted(destination)
	if err != nil || len(batches) != 1 {
		t.Fatalf("journal recovery result: batches=%d err=%v", len(batches), err)
	}
}

func TestActivateRejectsRestrictedClassification(t *testing.T) {
	root := t.TempDir()
	candidate := filepath.Join(root, "metadata.json")
	input := `{"schemaVersion":"mimir.metadata/v1","revision":"restricted-1","status":"draft","sources":[{"id":"orders","classification":"restricted","objects":[{"id":"orders","path":"orders.jsonl"}]}]}`
	if err := os.WriteFile(candidate, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	stateDir := filepath.Join(root, "state")
	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err == nil {
		t.Fatal("Activate() allowed restricted data in the local reference profile")
	}
	if _, err := os.Stat(filepath.Join(stateDir, "metadata")); !os.IsNotExist(err) {
		t.Fatalf("denied activation persisted metadata: %v", err)
	}
	audits, err := readAndRepairTail[auditEvent](filepath.Join(stateDir, "audit", "events.jsonl"))
	if err != nil || len(audits) != 1 || audits[0].Operation != "access.denied" {
		t.Fatalf("denied activation audit evidence: events=%+v err=%v", audits, err)
	}
}

func TestActivateRejectsChangedContentForAnExistingRevisionIdentity(t *testing.T) {
	root := t.TempDir()
	candidate := filepath.Join(root, "metadata.json")
	stateDir := filepath.Join(root, "state")
	if err := os.WriteFile(candidate, []byte(draftMetadata("immutable-1")), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(draftMetadata("immutable-1"), `"path":"orders.jsonl"`, `"path":"other.jsonl"`, 1)
	if err := os.WriteFile(candidate, []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err == nil {
		t.Fatal("Activate() changed content under an existing revision identity")
	}
}

func TestRunRejectsCheckpointWhenDestinationCommitIsMissing(t *testing.T) {
	_, sourceDir, stateDir, candidate := setupLocalRun(t, "durability-1", `{"id":10}`)
	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
		t.Fatal(err)
	}
	options := localRunOptions(sourceDir, stateDir)
	if _, err := Run(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(stateDir, "data", "destination.jsonl")); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(context.Background(), options); err == nil {
		t.Fatal("Run() recreated data missing from a previously committed checkpoint")
	}
}

func TestReadCommittedRejectsDuplicateJournalKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "destination.jsonl")
	entry := `{"batchId":"one","batchId":"two","metadataRevision":"rev","sourceDigest":"digest","records":[],"commitEvidence":"evidence"}` + "\n"
	if err := os.WriteFile(path, []byte(entry), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadCommitted(path); err == nil {
		t.Fatal("ReadCommitted() accepted duplicate journal keys")
	}
}

func TestReadCommittedRejectsUnknownJournalFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "destination.jsonl")
	entry := `{"batchId":"one","metadataRevision":"rev","sourceDigest":"digest","records":[],"commitEvidence":"evidence","unexpected":true}` + "\n"
	if err := os.WriteFile(path, []byte(entry), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadCommitted(path); err == nil {
		t.Fatal("ReadCommitted() accepted an unknown journal field")
	}
}

func TestReadCommittedRejectsPermissiveJournalPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "destination.jsonl")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadCommitted(path); err == nil {
		t.Fatal("ReadCommitted() accepted a journal readable by other users")
	}
}

func setupLocalRun(t *testing.T, revision, record string) (root, sourceDir, stateDir, candidate string) {
	t.Helper()
	root = t.TempDir()
	sourceDir = filepath.Join(root, "source")
	stateDir = filepath.Join(root, "state")
	candidate = filepath.Join(root, "metadata.json")
	if err := os.MkdirAll(sourceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(candidate, []byte(draftMetadata(revision)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "orders.jsonl"), []byte(record+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root, sourceDir, stateDir, candidate
}

func localRunOptions(sourceDir, stateDir string) Options {
	return Options{SourceRoot: sourceDir, StateDir: stateDir, Actor: "local-user", AllowLocal: true}
}

func draftMetadata(revision string) string {
	return `{"schemaVersion":"mimir.metadata/v1","revision":"` + revision + `","status":"draft","sources":[{"id":"orders","classification":"internal","objects":[{"id":"orders","path":"orders.jsonl"}]}]}`
}

func TestReplayFaultMatrixRestartsWithoutDuplicatingCommitOrAcknowledgement(t *testing.T) {
	stages := []string{
		"before_replay_plan_write",
		"after_replay_plan_write",
		"before_replay_audit",
		"after_replay_audit",
		"before_replay_run_reconcile",
		"after_replay_run_reconcile",
	}
	for _, stage := range stages {
		t.Run(stage, func(t *testing.T) {
			root, sourceDir, stateDir, candidate := setupLocalRun(t, "replay-fault-"+stage, `{"id":21}`)
			if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
				t.Fatal(err)
			}
			options := localRunOptions(sourceDir, stateDir)
			original, err := Run(context.Background(), options)
			if err != nil {
				t.Fatal(err)
			}
			injected := false
			func() {
				defer func() {
					crash, ok := recover().(injectedReplayCrash)
					if !ok || crash.stage != stage {
						t.Fatalf("replay did not stop at injected boundary %q: recovered=%v", stage, crash)
					}
				}()
				_, _ = replayWithFaultHook(context.Background(), options, original.RunID, "recovery", func(current string) bool {
					if current == stage {
						injected = true
						return true
					}
					return false
				})
			}()
			if !injected {
				t.Fatalf("replay fault boundary %q was not reached", stage)
			}
			replayed, err := Replay(context.Background(), options, original.RunID, "recovery")
			if err != nil || replayed.OriginalRunID != original.RunID || replayed.Result.RunID != original.RunID ||
				!replayed.Result.AlreadyCommitted {
				t.Fatalf("replay restart result=%+v err=%v", replayed, err)
			}
			batches, err := ReadCommitted(filepath.Join(root, "state", "data", "destination.jsonl"))
			if err != nil || len(batches) != 1 {
				t.Fatalf("restart destination commits=%d err=%v", len(batches), err)
			}
			checkpoints, err := readAndRepairTail[checkpoint](filepath.Join(stateDir, "state", "checkpoints.jsonl"))
			if err != nil || len(checkpoints) != 1 {
				t.Fatalf("restart checkpoints=%d err=%v", len(checkpoints), err)
			}
			audits, err := readAndRepairTail[auditEvent](filepath.Join(stateDir, "audit", "events.jsonl"))
			if err != nil {
				t.Fatal(err)
			}
			prepared := 0
			for _, event := range audits {
				if event.Operation == "replay.prepared" {
					prepared++
				}
			}
			if prepared != 1 {
				t.Fatalf("replay preparation audit count=%d, want 1", prepared)
			}
		})
	}
}
