package domain

import (
	"context"
	"testing"
)

func TestUnknownCommitRequiresReconciliationAndIsNotBlindRetrySafe(t *testing.T) {
	failure := NewFailure(FailureAmbiguousCommit, "destination.commit", "outcome uncertain")
	if failure.CommitCertainty != CommitUnknown || failure.Reconciliation != "reconcile_before_retry" {
		t.Fatalf("ambiguous failure omitted reconciliation: %+v", failure)
	}

	if failure.BlindRetrySafe() {
		t.Fatal("ambiguous commit was marked safe for blind retry")
	}
}

func TestContextFailureDistinguishesCancellationAndDeadline(t *testing.T) {
	cancelled := ContextFailure("source.read", context.Canceled)
	if cancelled.Category != FailureCancelled || cancelled.Operation != "source.read" {
		t.Fatalf("cancelled context failure=%+v", cancelled)
	}
	timedOut := ContextFailure("source.read", context.DeadlineExceeded)
	if timedOut.Category != FailureTimeout || timedOut.Operation != "source.read" {
		t.Fatalf("deadline context failure=%+v", timedOut)
	}
}
