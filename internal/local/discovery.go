package local

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"time"

	"github.com/kumerian/mimir/internal/discovery"
	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/metadata"
	"github.com/kumerian/mimir/internal/ports"
)

type DiscoveryOptions struct {
	SourceRoot         string
	AllowedSourceRoot  string
	StateDir           string
	Actor              string
	AllowLocal         bool
	Disposition        string
	MetadataRepository ports.MetadataRepository
	ProposalStore      ports.MetadataProposalStore
}

type DiscoveryResult struct {
	ProposalID string             `json:"proposalId"`
	Proposal   discovery.Proposal `json:"proposal"`
}

// Discover observes only JSON field names and types, persists a proposal, and
// never changes the active metadata pointer.
func Discover(ctx context.Context, options DiscoveryOptions) (DiscoveryResult, error) {
	if err := ctx.Err(); err != nil {
		return DiscoveryResult{}, domain.ContextFailure("discovery.start", err)
	}
	if !options.AllowLocal || options.Actor != "local-user" {
		if err := recordDenial(options.StateDir, options.Actor, "metadata.discover", "access_denied"); err != nil {
			return DiscoveryResult{}, domain.NewFailure(domain.FailureUnavailable, "audit.denial", "discovery was denied and denial audit could not be persisted")
		}
		return DiscoveryResult{}, domain.NewFailure(domain.FailureAccessDenied, "metadata.discover", "explicit local-user authorization is required")
	}
	if options.SourceRoot == "" || options.StateDir == "" {
		return DiscoveryResult{}, domain.NewFailure(domain.FailureInvalidInput, "discovery.plan", "source root and state directory are required")
	}
	if !isPrivateDirectory(options.StateDir) {
		return DiscoveryResult{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.active", "private local metadata directory is unavailable")
	}
	lock, err := acquireLock(options.StateDir)
	if err != nil {
		return DiscoveryResult{}, err
	}
	defer releaseLock(lock)
	parsed, _, err := loadActiveMetadataWithRepository(ctx, options.StateDir, options.MetadataRepository)
	if err != nil {
		return DiscoveryResult{}, err
	}
	if err := localMetadataPolicy(parsed.Document); err != nil {
		if err := recordDenial(options.StateDir, options.Actor, "metadata.discover", "access_denied"); err != nil {
			return DiscoveryResult{}, domain.NewFailure(domain.FailureUnavailable, "audit.denial", "discovery was denied and denial audit could not be persisted")
		}
		return DiscoveryResult{}, domain.NewFailure(domain.FailureAccessDenied, "discovery.plan", "local reference profile permits only public or internal metadata")
	}
	if err := validateLocalMetadataCapabilities(parsed.Document); err != nil {
		return DiscoveryResult{}, err
	}
	if len(parsed.Document.Sources) != 1 || len(parsed.Document.Sources[0].Objects) != 1 {
		return DiscoveryResult{}, domain.NewFailure(domain.FailureUnsupportedCapability, "discovery.plan", "local discovery requires exactly one source and one object")
	}
	source, object := parsed.Document.Sources[0], parsed.Document.Sources[0].Objects[0]
	provider := jsonlDiscoveryProvider{root: options.SourceRoot, allowedRoot: options.AllowedSourceRoot}
	rawObservation, err := observeWithProvider(ctx, provider, object.Path)
	if err != nil {
		return DiscoveryResult{}, err
	}
	var observedSource observedSchema
	if err := decodeStrict(rawObservation, &observedSource); err != nil {
		return DiscoveryResult{}, domain.NewFailure(domain.FailureInternal, "discovery.observe", "provider returned an invalid normalized observation")
	}
	observed := observedSource.Columns
	priorClasses := make(map[string]string, len(object.Columns))
	for _, column := range object.Columns {
		priorClasses[column.ID] = column.Classification
	}
	for i := range observed {
		observed[i].Classification = priorClasses[observed[i].ID]
	}
	report, err := discovery.Classify(object.Columns, observed, options.Disposition)
	if err != nil {
		return DiscoveryResult{}, err
	}
	observation := discovery.Observation{
		ContractVersion: discovery.ContractVersion, Provider: "local-jsonl",
		ProviderVersion: "local.discovery/v1", Scope: source.ID + "/" + object.ID,
		SourceDigest: observedSource.SourceDigest, ObservedAt: time.Now().UTC(), Columns: observed,
	}
	proposal, err := discovery.CreateProposal(parsed.Document.Revision, observation, report)
	if err != nil {
		return DiscoveryResult{}, err
	}
	proposalBytes, err := metadata.CanonicalJSON(proposal)
	if err != nil {
		return DiscoveryResult{}, domain.NewFailure(domain.FailureInternal, "discovery.persist", "discovery proposal cannot be encoded")
	}
	proposalID := metadata.Digest(proposalBytes)
	proposalStore := options.ProposalStore
	if proposalStore == nil {
		proposalStore = fileMetadataProposalStore{stateDir: options.StateDir}
	}
	if err := proposalStore.Append(ctx, proposalID, proposalBytes); err != nil {
		return DiscoveryResult{}, err
	}
	event := auditEvent{
		EventID:   metadata.Digest([]byte("metadata.discovered\x00" + proposalID)),
		Operation: "metadata.discovered", Actor: options.Actor, MetadataRevision: parsed.Document.Revision,
		PlanID: proposalID, RecordCount: observedSource.RecordCount, OccurredAt: observation.ObservedAt.Format(time.RFC3339Nano),
	}
	if err := ensureAudit(filepath.Join(options.StateDir, "audit", "events.jsonl"), event); err != nil {
		return DiscoveryResult{}, domain.NewFailure(domain.FailureUnavailable, "discovery.audit", "proposal is stored but its audit event is pending")
	}
	return DiscoveryResult{ProposalID: proposalID, Proposal: proposal}, nil
}

func observeWithProvider(ctx context.Context, provider ports.DiscoveryProvider, path string) (json.RawMessage, error) {
	capabilities := provider.Capabilities()
	if capabilities.ContractVersion != ports.DiscoveryProviderVersion ||
		!slices.Contains(capabilities.Operations, "bounded_jsonl_schema_observation") {
		return nil, domain.NewFailure(domain.FailureUnsupportedCapability, "discovery.plan", "provider lacks the required versioned schema-observation capability")
	}
	return provider.Observe(ctx, path)
}

func inferColumns(records [][]byte) ([]metadata.Column, error) {
	types := make(map[string]string)
	for _, record := range records {
		decoder := json.NewDecoder(bytes.NewReader(record))
		decoder.UseNumber()
		var values map[string]any
		if err := decoder.Decode(&values); err != nil || values == nil {
			return nil, domain.NewFailure(domain.FailureInvalidInput, "discovery.observe", "record is not a JSON object")
		}
		for field, value := range values {
			kind, err := observedType(value)
			if err != nil {
				return nil, err
			}
			if prior, ok := types[field]; ok && prior != kind {
				return nil, domain.NewFailure(domain.FailureConflict, "discovery.observe", "field has inconsistent types across the source snapshot")
			}
			types[field] = kind
		}
	}
	fields := make([]string, 0, len(types))
	for field := range types {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	columns := make([]metadata.Column, 0, len(fields))
	for _, field := range fields {
		columns = append(columns, metadata.Column{ID: field, Type: types[field]})
	}
	return columns, nil
}

func observedType(value any) (string, error) {
	switch typed := value.(type) {
	case string:
		return "string", nil
	case bool:
		return "boolean", nil
	case json.Number:
		if _, err := strconv.ParseInt(typed.String(), 10, 64); err == nil {
			return "integer", nil
		}
		if _, err := strconv.ParseFloat(typed.String(), 64); err == nil {
			return "number", nil
		}
		return "", domain.NewFailure(domain.FailureInvalidInput, "discovery.observe", "numeric field cannot be classified")
	case nil:
		return "", domain.NewFailure(domain.FailureUnsupportedCapability, "discovery.observe", "null-only values do not provide a schema type")
	case map[string]any, []any:
		return "", domain.NewFailure(domain.FailureUnsupportedCapability, "discovery.observe", "nested JSON values are not supported by local schema discovery")
	default:
		return "", domain.NewFailure(domain.FailureUnsupportedCapability, "discovery.observe", "field type is not supported by local schema discovery")
	}
}
