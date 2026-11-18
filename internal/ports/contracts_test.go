package ports

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kumerian/mimir/internal/domain"
)

type sideEffectAdapter struct {
	called int
}

func (a *sideEffectAdapter) Capabilities() DestinationCapabilities {
	return DestinationCapabilities{Version: "dest/v1", IdempotentWrites: true, DurableCommitEvidence: true}
}

func (a *sideEffectAdapter) CommitBatch(context.Context, string, json.RawMessage) (string, error) {
	a.called++
	return "evidence", nil
}

func (a *sideEffectAdapter) Reconcile(context.Context, string, json.RawMessage) (bool, error) {
	a.called++
	return false, nil
}

func TestBoundedRunCapabilityRejectionDoesNotCallDestination(t *testing.T) {
	destination := &sideEffectAdapter{}
	source := SourceCapabilities{Version: "source/v1", BoundedSnapshot: true, StableIdentity: true}
	err := ValidateBoundedRun(source, destination.Capabilities())
	failure, ok := err.(domain.Failure)
	if !ok || failure.Category != domain.FailureUnsupportedCapability {
		t.Fatalf("ValidateBoundedRun() error=%v, want stable unsupported capability", err)
	}
	if destination.called != 0 {
		t.Fatalf("capability rejection made %d destination calls", destination.called)
	}
}

func TestBoundedRunRejectsIncompatibleBatchLimits(t *testing.T) {
	err := ValidateBoundedRun(
		SourceCapabilities{Version: "source/v1", BoundedSnapshot: true, StableIdentity: true, MaximumBatchRecords: 500},
		DestinationCapabilities{Version: "destination/v1", IdempotentWrites: true, DurableCommitEvidence: true, Reconcile: true, MaximumBatchRecords: 100},
	)
	if err == nil {
		t.Fatal("incompatible source/destination batch limits were accepted")
	}
}
