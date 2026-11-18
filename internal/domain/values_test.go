package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestIdentifierConstructorAcceptsBoundariesAndRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		kind    IdentifierKind
		value   string
		wantErr bool
	}{
		{name: "shortest", kind: RunID, value: "r"},
		{name: "maximum", kind: PlanID, value: "a" + strings.Repeat("b", 127)},
		{name: "empty", kind: RunID, value: "", wantErr: true},
		{name: "too long", kind: BatchID, value: "a" + strings.Repeat("b", 128), wantErr: true},
		{name: "path", kind: PlanID, value: "../secret", wantErr: true},
		{name: "unknown kind", kind: "filesystem_path", value: "safe", wantErr: true},
		{name: "adapter version", kind: AdapterVersionID, value: "local.source/v1"},
		{name: "adapter version traversal", kind: AdapterVersionID, value: "local/../v1", wantErr: true},
		{name: "source identifier", kind: SourceID, value: "orders-source"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := NewIdentifier(tt.kind, tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewIdentifier() = %#v, %v; wantErr=%v", value, err, tt.wantErr)
			}
			if err == nil && (value.Kind() != tt.kind || value.String() != tt.value) {
				t.Fatalf("identifier changed value or kind: %+v", value)
			}
			if err != nil && !IsFailure(err, FailureInvalidInput) {
				t.Fatalf("invalid identifier error=%#v, want stable invalid_input", err)
			}
		})
	}
}

func TestAttemptConstructorRequiresValidIdentityAndPositiveIndex(t *testing.T) {
	attempt, err := NewAttempt("run-1", "plan-1", 1)
	if err != nil || attempt.Run.String() != "run-1" || attempt.Plan.String() != "plan-1" || attempt.Index != 1 {
		t.Fatalf("valid attempt=%+v err=%v", attempt, err)
	}
	for _, invalid := range []struct {
		run, plan string
		index     int
	}{
		{run: "../run", plan: "plan-1", index: 1},
		{run: "run-1", plan: "../plan", index: 1},
		{run: "run-1", plan: "plan-1", index: 0},
	} {
		if _, err := NewAttempt(invalid.run, invalid.plan, invalid.index); !IsFailure(err, FailureInvalidInput) {
			t.Fatalf("NewAttempt(%+v) error=%#v, want stable invalid_input", invalid, err)
		}
	}
}

func TestCheckpointRequiresDurableCommitEvidence(t *testing.T) {
	valid, err := NewCheckpoint("run-1", "plan-1", "batch-1", 1, "commit-1")
	if err != nil || valid.ContractVersion != CheckpointContractVersion ||
		valid.CommitEvidence.Kind() != CommitEvidenceID {
		t.Fatalf("valid checkpoint=%+v err=%v", valid, err)
	}
	for _, tt := range []struct {
		name, run, plan, batch, evidence string
		attempt                          int
	}{
		{name: "missing commit evidence", run: "run-1", plan: "plan-1", batch: "batch-1", attempt: 1},
		{name: "missing attempt", run: "run-1", plan: "plan-1", batch: "batch-1", evidence: "commit-1"},
		{name: "invalid batch", run: "run-1", plan: "plan-1", batch: "../batch", evidence: "commit-1", attempt: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCheckpoint(tt.run, tt.plan, tt.batch, tt.attempt, tt.evidence)
			if !IsFailure(err, FailureInvalidInput) {
				t.Fatalf("NewCheckpoint() error=%#v, want stable invalid_input", err)
			}
		})
	}
}

func TestAdapterFailureMapsCategoryAndCauseReferenceWithoutLeakingCause(t *testing.T) {
	cause := errors.New("adapter failed with synthetic-secret-canary")
	failure := MapAdapterFailure(FailureUnavailable, "destination.commit", "destination operation failed", cause)
	if failure.Category != FailureUnavailable || failure.Operation != "destination.commit" ||
		!strings.HasPrefix(failure.CauseReference, "sha256:") {
		t.Fatalf("adapter failure mapping=%+v", failure)
	}
	serialized, err := json.Marshal(failure)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(failure.Error(), "synthetic-secret-canary") ||
		strings.Contains(string(serialized), "synthetic-secret-canary") {
		t.Fatalf("adapter cause leaked through stable failure: %s", serialized)
	}
	repeated := MapAdapterFailure(FailureUnavailable, "destination.commit", "destination operation failed", cause)
	if repeated.CauseReference != failure.CauseReference {
		t.Fatal("cause reference is not deterministic for the same adapter failure")
	}
}
