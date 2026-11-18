package discovery

import (
	"fmt"
	"sort"
	"time"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/metadata"
)

const ContractVersion = "mimir.discovery/v1"

type Observation struct {
	ContractVersion string            `json:"contractVersion"`
	Provider        string            `json:"provider"`
	ProviderVersion string            `json:"providerVersion"`
	Scope           string            `json:"scope"`
	SourceDigest    string            `json:"sourceDigest"`
	ObservedAt      time.Time         `json:"observedAt"`
	Columns         []metadata.Column `json:"columns"`
}

type Change struct {
	Field  string `json:"field"`
	Kind   string `json:"kind"`
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
}

type Report struct {
	ContractVersion string   `json:"contractVersion"`
	Compatible      bool     `json:"compatible"`
	Disposition     string   `json:"disposition"`
	Changes         []Change `json:"changes"`
}

type Proposal struct {
	ActiveRevision string            `json:"activeRevision"`
	Observation    Observation       `json:"observation"`
	Report         Report            `json:"report"`
	Proposed       []metadata.Column `json:"proposedColumns"`
	ExecutionReady bool              `json:"executionReady"`
}

func Classify(active, observed []metadata.Column, disposition string) (Report, error) {
	switch disposition {
	case "reject", "hold", "quarantine", "continue":
	default:
		return Report{}, domain.NewFailure(domain.FailureInvalidInput, "discovery.policy", "explicit reject, hold, quarantine, or continue policy is required")
	}
	activeByID, err := columnMap(active)
	if err != nil {
		return Report{}, err
	}
	observedByID, err := columnMap(observed)
	if err != nil {
		return Report{}, err
	}
	changes := make([]Change, 0)
	compatible := true
	for id, column := range observedByID {
		prior, exists := activeByID[id]
		if !exists {
			changes = append(changes, Change{Field: id, Kind: "added", After: column.Type})
			continue
		}
		if prior.Type != column.Type {
			changes = append(changes, Change{Field: id, Kind: "type_changed", Before: prior.Type, After: column.Type})
			compatible = false
		}
		if prior.Classification != column.Classification {
			changes = append(changes, Change{Field: id, Kind: "classification_changed", Before: prior.Classification, After: column.Classification})
			compatible = false
		}
	}
	for id, column := range activeByID {
		if _, exists := observedByID[id]; !exists {
			changes = append(changes, Change{Field: id, Kind: "removed", Before: column.Type})
			compatible = false
		}
	}
	sort.Slice(changes, func(i, j int) bool {
		if changes[i].Field == changes[j].Field {
			return changes[i].Kind < changes[j].Kind
		}
		return changes[i].Field < changes[j].Field
	})
	if disposition == "continue" && !compatible {
		return Report{}, domain.NewFailure(domain.FailureInvalidInput, "discovery.policy", "continue is allowed only for compatible additive drift")
	}
	return Report{ContractVersion: ContractVersion, Compatible: compatible, Disposition: disposition, Changes: changes}, nil
}

func CreateProposal(activeRevision string, observation Observation, report Report) (Proposal, error) {
	if activeRevision == "" || observation.ContractVersion != ContractVersion || observation.Provider == "" ||
		observation.ProviderVersion == "" || observation.Scope == "" || observation.SourceDigest == "" || observation.ObservedAt.IsZero() {
		return Proposal{}, domain.NewFailure(domain.FailureInvalidInput, "discovery.proposal", "active revision and complete observation provenance are required")
	}
	if _, err := columnMap(observation.Columns); err != nil {
		return Proposal{}, err
	}
	columns := append([]metadata.Column(nil), observation.Columns...)
	return Proposal{
		ActiveRevision: activeRevision, Observation: observation, Report: report,
		Proposed: columns, ExecutionReady: false,
	}, nil
}

func columnMap(columns []metadata.Column) (map[string]metadata.Column, error) {
	result := make(map[string]metadata.Column, len(columns))
	for _, column := range columns {
		if column.ID == "" || result[column.ID].ID != "" {
			return nil, domain.NewFailure(domain.FailureInvalidInput, "discovery.schema", "observed column names must be non-empty and unique")
		}
		switch column.Type {
		case "string", "number", "integer", "boolean":
		default:
			return nil, domain.NewFailure(domain.FailureInvalidInput, "discovery.schema", fmt.Sprintf("unsupported type for observed field %q", column.ID))
		}
		result[column.ID] = column
	}
	return result, nil
}
