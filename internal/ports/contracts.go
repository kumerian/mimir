// Package ports declares application-owned capabilities and adapter contracts.
// Implementations belong outside the domain/application dependency direction.
package ports

import (
	"context"
	"encoding/json"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/metadata"
)

const (
	MetadataRepositoryVersion      = "mimir.metadata-repository/v1"
	MetadataValidationStoreVersion = "mimir.metadata-validation-store/v1"
	DiscoveryProviderVersion       = "mimir.discovery-provider/v1"
	SourceAdapterVersion           = "mimir.source-adapter/v1"
	DestinationAdapterVersion      = "mimir.destination-adapter/v1"
	SchedulerVersion               = "mimir.scheduler/v1"
	RunStateStoreVersion           = "mimir.run-state-store/v1"
	SecretResolverVersion          = "mimir.secret-resolver/v1"
	TransformEngineVersion         = "mimir.transform-engine/v1"
	QualityEngineVersion           = "mimir.quality-engine/v1"
	CleaningEngineVersion          = "mimir.cleaning-engine/v1"
	AuditStoreVersion              = "mimir.audit-store/v1"
	TelemetryVersion               = "mimir.telemetry/v1"
)

type SourceCapabilities struct {
	Version             string
	BoundedSnapshot     bool
	StableIdentity      bool
	SchemaObservation   bool
	Acknowledgement     bool
	MaximumBatchRecords int
}

type DestinationCapabilities struct {
	Version               string
	IdempotentWrites      bool
	DurableCommitEvidence bool
	Reconcile             bool
	MaximumBatchRecords   int
}

type CapabilitySet struct {
	ContractVersion string
	Operations      []string
	Unsupported     []string
	MaximumBytes    int64
	Concurrency     int
	Classifications []string
}

func ValidateBoundedRun(source SourceCapabilities, destination DestinationCapabilities) error {
	if _, err := domain.NewIdentifier(domain.AdapterVersionID, source.Version); err != nil ||
		!source.BoundedSnapshot || !source.StableIdentity {
		return domain.NewFailure(domain.FailureUnsupportedCapability, "plan.capabilities", "source must declare a versioned bounded snapshot with stable identity")
	}
	if _, err := domain.NewIdentifier(domain.AdapterVersionID, destination.Version); err != nil ||
		!destination.IdempotentWrites ||
		!destination.DurableCommitEvidence || !destination.Reconcile {
		return domain.NewFailure(domain.FailureUnsupportedCapability, "plan.capabilities", "destination must declare idempotent writes, durable commit evidence, and reconciliation")
	}
	if source.MaximumBatchRecords > 0 && destination.MaximumBatchRecords > 0 &&
		source.MaximumBatchRecords > destination.MaximumBatchRecords {
		return domain.NewFailure(domain.FailureUnsupportedCapability, "plan.capabilities", "source batch limit exceeds destination capability")
	}
	return nil
}

type MetadataRepository interface {
	Capabilities() CapabilitySet
	AppendDraft(context.Context, json.RawMessage, string) error
	List(context.Context) ([]json.RawMessage, error)
	LoadActive(context.Context) (json.RawMessage, error)
	Activate(context.Context, json.RawMessage, string) (string, error)
}

type MetadataValidationStore interface {
	Capabilities() CapabilitySet
	Append(context.Context, metadata.ValidationReport) error
	Load(context.Context, string) (metadata.ValidationReport, error)
}

type DiscoveryProvider interface {
	Capabilities() CapabilitySet
	Observe(context.Context, string) (json.RawMessage, error)
}

type SourceAdapter interface {
	Capabilities() SourceCapabilities
	ReadBatch(context.Context, string) ([]json.RawMessage, string, error)
	Acknowledge(context.Context, string) error
}

type DestinationAdapter interface {
	Capabilities() DestinationCapabilities
	CommitBatch(context.Context, string, json.RawMessage) (string, error)
	Reconcile(context.Context, string, json.RawMessage) (bool, error)
}

type Scheduler interface {
	Capabilities() CapabilitySet
	Dispatch(context.Context, json.RawMessage) error
}

type RunStateStore interface {
	Capabilities() CapabilitySet
	AppendTransition(context.Context, string, json.RawMessage) error
	LoadHistory(context.Context, string) ([]json.RawMessage, error)
}

type SecretResolver interface {
	Capabilities() CapabilitySet
	// WithSecret invokes use synchronously with a temporary secret buffer and a
	// bounded context. The callback must honor its context, must not retain or
	// copy the value, and must not serialize it to metadata or observability.
	// Implementations clear their temporary buffer before returning.
	WithSecret(context.Context, string, string, func(context.Context, []byte) error) error
}

type TransformEngine interface {
	Capabilities() CapabilitySet
	Transform(context.Context, json.RawMessage, []json.RawMessage) (json.RawMessage, error)
}

type QualityEngine interface {
	Capabilities() CapabilitySet
	Evaluate(context.Context, json.RawMessage, []json.RawMessage) (json.RawMessage, error)
}

type CleaningEngine interface {
	Capabilities() CapabilitySet
	Clean(context.Context, json.RawMessage, json.RawMessage, string) (json.RawMessage, error)
}

type AuditStore interface {
	Capabilities() CapabilitySet
	Append(context.Context, json.RawMessage) error
}

type Telemetry interface {
	Capabilities() CapabilitySet
	Emit(context.Context, json.RawMessage) error
}
