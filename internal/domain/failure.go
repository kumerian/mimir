package domain

import (
	"context"
	"errors"
	"fmt"
)

const FailureContractVersion = "mimir.failure/v1"

type FailureCategory string

const (
	FailureInvalidInput          FailureCategory = "invalid_input"
	FailureUnsupportedCapability FailureCategory = "unsupported_capability"
	FailureAccessDenied          FailureCategory = "access_denied"
	FailureConflict              FailureCategory = "conflict"
	FailurePolicyRejected        FailureCategory = "policy_rejected"
	FailureUnavailable           FailureCategory = "unavailable"
	FailureAmbiguousCommit       FailureCategory = "ambiguous_commit"
	FailureCancelled             FailureCategory = "cancelled"
	FailureTimeout               FailureCategory = "timeout"
	FailureInternal              FailureCategory = "internal"
)

type CommitCertainty string

const (
	CommitNotCommitted CommitCertainty = "not_committed"
	CommitCommitted    CommitCertainty = "committed"
	CommitUnknown      CommitCertainty = "unknown"
)

// Failure is the stable, technology-neutral error envelope crossing Mimir
// module boundaries. Cause contains a safe reference, never an adapter payload.
type Failure struct {
	ContractVersion string          `json:"contractVersion"`
	Category        FailureCategory `json:"category"`
	Operation       string          `json:"operation"`
	Stage           string          `json:"stage,omitempty"`
	Retryable       bool            `json:"retryable"`
	CommitCertainty CommitCertainty `json:"commitCertainty"`
	Reconciliation  string          `json:"reconciliation,omitempty"`
	CauseReference  string          `json:"causeReference,omitempty"`
	SafeDiagnostic  string          `json:"safeDiagnostic"`
}

func NewFailure(category FailureCategory, operation, diagnostic string) Failure {
	failure := Failure{
		ContractVersion: FailureContractVersion,
		Category:        category,
		Operation:       operation,
		CommitCertainty: CommitNotCommitted,
		SafeDiagnostic:  diagnostic,
	}
	if category == FailureAmbiguousCommit {
		failure.CommitCertainty = CommitUnknown
		failure.Reconciliation = "reconcile_before_retry"
	}
	return failure
}

func ContextFailure(operation string, err error) Failure {
	if errors.Is(err, context.DeadlineExceeded) {
		return NewFailure(FailureTimeout, operation, "operation deadline exceeded")
	}
	return NewFailure(FailureCancelled, operation, "operation was cancelled")
}

func (f Failure) Error() string {
	if f.Category == FailureAmbiguousCommit || f.CommitCertainty == CommitUnknown {
		return fmt.Sprintf("%s: %s (reconciliation required)", f.Category, f.SafeDiagnostic)
	}
	return fmt.Sprintf("%s: %s", f.Category, f.SafeDiagnostic)
}

func (f Failure) BlindRetrySafe() bool {
	return f.CommitCertainty == CommitNotCommitted && f.Category != FailureAmbiguousCommit
}
