package local

import (
	"context"
	"encoding/json"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/metadata"
	"github.com/kumerian/mimir/internal/ports"
)

type jsonlDiscoveryProvider struct {
	root string
}

type observedSchema struct {
	SourceDigest string            `json:"sourceDigest"`
	RecordCount  int               `json:"recordCount"`
	Columns      []metadata.Column `json:"columns"`
}

func (p jsonlDiscoveryProvider) Capabilities() ports.CapabilitySet {
	return ports.CapabilitySet{
		ContractVersion: ports.DiscoveryProviderVersion,
		Operations:      []string{"bounded_jsonl_schema_observation"},
		Unsupported:     []string{"nested_values", "null_only_fields", "active_metadata_mutation"},
		MaximumBytes:    maxSourceBytes,
		Concurrency:     1,
		Classifications: []string{"public", "internal"},
	}
}

func (p jsonlDiscoveryProvider) Observe(ctx context.Context, relativePath string) (json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, domain.ContextFailure("discovery.observe", err)
	}
	if p.root == "" || relativePath == "" {
		return nil, domain.NewFailure(domain.FailureInvalidInput, "discovery.observe", "source root and object path are required")
	}
	path, err := safeSourcePath(p.root, relativePath)
	if err != nil {
		return nil, err
	}
	records, snapshot, err := readSnapshot(ctx, path)
	if err != nil {
		return nil, err
	}
	recordBytes := make([][]byte, len(records))
	for i := range records {
		recordBytes[i] = records[i]
	}
	columns, err := inferColumns(recordBytes)
	if err != nil {
		return nil, err
	}
	observation := observedSchema{SourceDigest: digest(snapshot), RecordCount: len(records), Columns: columns}
	raw, err := json.Marshal(observation)
	if err != nil {
		return nil, domain.NewFailure(domain.FailureInternal, "discovery.observe", "schema observation cannot be encoded")
	}
	return raw, nil
}

var _ ports.DiscoveryProvider = jsonlDiscoveryProvider{}
