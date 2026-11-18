package replay

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/metadata"
)

const ContractVersion = "mimir.replay/v1"

type Plan struct {
	RunID            string   `json:"runId"`
	MetadataRevision string   `json:"metadataRevision"`
	MetadataDigest   string   `json:"metadataDigest"`
	SourceDigest     string   `json:"sourceDigest"`
	AdapterVersions  []string `json:"adapterVersions"`
	RuleVersions     []string `json:"ruleVersions"`
}

type ReplayPlan struct {
	ContractVersion  string   `json:"contractVersion"`
	RunID            string   `json:"runId"`
	OriginalRunID    string   `json:"originalRunId"`
	Reason           string   `json:"reason"`
	MetadataRevision string   `json:"metadataRevision"`
	MetadataDigest   string   `json:"metadataDigest"`
	SourceDigest     string   `json:"sourceDigest"`
	AdapterVersions  []string `json:"adapterVersions"`
	RuleVersions     []string `json:"ruleVersions"`
}

func Prepare(original, current Plan, reason string) (ReplayPlan, error) {
	switch strings.TrimSpace(reason) {
	case "operator_request", "recovery", "incident":
	default:
		return ReplayPlan{}, domain.NewFailure(domain.FailureInvalidInput, "replay.prepare", "a supported replay reason is required")
	}
	if original.RunID == "" || original.MetadataRevision == "" || original.MetadataDigest == "" ||
		original.SourceDigest == "" || len(original.AdapterVersions) == 0 {
		return ReplayPlan{}, domain.NewFailure(domain.FailureInvalidInput, "replay.prepare", "original plan does not contain all required pinned inputs")
	}
	if original.MetadataRevision != current.MetadataRevision ||
		original.MetadataDigest != current.MetadataDigest ||
		original.SourceDigest != current.SourceDigest ||
		!equal(original.AdapterVersions, current.AdapterVersions) ||
		!equal(original.RuleVersions, current.RuleVersions) {
		return ReplayPlan{}, domain.NewFailure(domain.FailureUnsupportedCapability, "replay.prepare", "pinned source, metadata, adapter, or rule input is unavailable or changed")
	}
	plan := ReplayPlan{
		ContractVersion: ContractVersion, OriginalRunID: original.RunID, Reason: strings.TrimSpace(reason),
		MetadataRevision: original.MetadataRevision, MetadataDigest: original.MetadataDigest,
		SourceDigest: original.SourceDigest, AdapterVersions: append([]string(nil), original.AdapterVersions...),
		RuleVersions: append([]string(nil), original.RuleVersions...),
	}
	identity, err := json.Marshal(plan)
	if err != nil {
		return ReplayPlan{}, domain.NewFailure(domain.FailureInternal, "replay.prepare", "replay identity cannot be created")
	}
	plan.RunID = metadata.Digest(identity)
	return plan, nil
}

func equal(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func (p ReplayPlan) Validate() error {
	if p.ContractVersion != ContractVersion || p.RunID == "" || p.OriginalRunID == "" {
		return fmt.Errorf("invalid replay plan contract")
	}
	return nil
}
