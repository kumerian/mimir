package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type IdentifierKind string

const (
	MetadataRevisionID IdentifierKind = "metadata_revision"
	SourceID           IdentifierKind = "source"
	ObjectID           IdentifierKind = "object"
	ColumnID           IdentifierKind = "column"
	RuleID             IdentifierKind = "rule"
	PlanID             IdentifierKind = "plan"
	RunID              IdentifierKind = "run"
	AttemptID          IdentifierKind = "attempt"
	BatchID            IdentifierKind = "batch"
	CommitEvidenceID   IdentifierKind = "commit_evidence"
	AdapterVersionID   IdentifierKind = "adapter_version"
	SecretReferenceID  IdentifierKind = "secret_reference"
	LineageID          IdentifierKind = "lineage"
)

var identifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
var adapterVersionPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{0,127}$`)

// Identifier is an opaque, validated domain identity.
type Identifier struct {
	kind  IdentifierKind
	value string
}

func NewIdentifier(kind IdentifierKind, value string) (Identifier, error) {
	switch kind {
	case MetadataRevisionID, SourceID, ObjectID, ColumnID, RuleID, PlanID, RunID,
		AttemptID, BatchID, CommitEvidenceID, SecretReferenceID, LineageID:
	default:
		if kind != AdapterVersionID {
			return Identifier{}, NewFailure(FailureInvalidInput, "domain.identifier", "identifier kind is unsupported")
		}
	}
	valid := identifierPattern.MatchString(value)
	if kind == AdapterVersionID {
		valid = adapterVersionPattern.MatchString(value) && !strings.Contains(value, "..") &&
			!strings.Contains(value, "//") && !strings.HasSuffix(value, "/")
	}
	if !valid {
		return Identifier{}, NewFailure(FailureInvalidInput, "domain.identifier", "identifier must contain 1-128 safe characters")
	}
	return Identifier{kind: kind, value: value}, nil
}

func (i Identifier) Kind() IdentifierKind { return i.kind }

func (i Identifier) String() string { return i.value }

type Attempt struct {
	Run   Identifier
	Plan  Identifier
	Index int
}

func NewAttempt(runID, planID string, index int) (Attempt, error) {
	run, err := NewIdentifier(RunID, runID)
	if err != nil {
		return Attempt{}, err
	}
	plan, err := NewIdentifier(PlanID, planID)
	if err != nil || index < 1 {
		return Attempt{}, NewFailure(FailureInvalidInput, "domain.attempt", "attempt requires valid run, plan, and positive index")
	}
	return Attempt{Run: run, Plan: plan, Index: index}, nil
}

// Checkpoint is valid only when it names durable commit evidence for the
// corresponding immutable run, plan, batch, and attempt.
type Checkpoint struct {
	ContractVersion string     `json:"contractVersion"`
	Run             Identifier `json:"run"`
	Plan            Identifier `json:"plan"`
	Batch           Identifier `json:"batch"`
	Attempt         int        `json:"attempt"`
	CommitEvidence  Identifier `json:"commitEvidence"`
}

const CheckpointContractVersion = "mimir.checkpoint/v1"

func NewCheckpoint(runID, planID, batchID string, attempt int, commitEvidence string) (Checkpoint, error) {
	run, err := NewIdentifier(RunID, runID)
	if err != nil {
		return Checkpoint{}, err
	}
	plan, err := NewIdentifier(PlanID, planID)
	if err != nil {
		return Checkpoint{}, err
	}
	batch, err := NewIdentifier(BatchID, batchID)
	if err != nil {
		return Checkpoint{}, err
	}
	evidence, err := NewIdentifier(CommitEvidenceID, commitEvidence)
	if err != nil || attempt < 1 {
		return Checkpoint{}, NewFailure(FailureInvalidInput, "domain.checkpoint", "attempt and durable commit evidence are required")
	}
	return Checkpoint{
		ContractVersion: CheckpointContractVersion,
		Run:             run,
		Plan:            plan,
		Batch:           batch,
		Attempt:         attempt,
		CommitEvidence:  evidence,
	}, nil
}

// MapAdapterFailure converts a technology error to a stable Mimir envelope.
// The raw error is never wrapped or serialized; its digest is a correlation
// reference suitable for matching a separately protected adapter log.
func MapAdapterFailure(category FailureCategory, operation, diagnostic string, cause error) Failure {
	failure := NewFailure(category, operation, diagnostic)
	if cause == nil {
		return failure
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%T\x00%s\x00%s", cause, operation, cause.Error())))
	failure.CauseReference = "sha256:" + hex.EncodeToString(sum[:])
	return failure
}

// IsFailure reports whether err is a Mimir failure without exposing an
// adapter-owned error to callers.
func IsFailure(err error, category FailureCategory) bool {
	var failure Failure
	return errors.As(err, &failure) && failure.Category == category
}
