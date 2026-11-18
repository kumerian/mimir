package planning

import (
	"fmt"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/metadata"
	"github.com/kumerian/mimir/internal/ports"
)

const (
	RunPlanContractVersion = "mimir.run-plan/v1"
	RecordContractVersion  = "mimir.record/json-object/v1"
)

type SourceCapabilities = ports.SourceCapabilities

type DestinationCapabilities = ports.DestinationCapabilities

type Request struct {
	SourceID     string
	ObjectID     string
	Authorized   bool
	MaximumClass string
	Source       SourceCapabilities
	Destination  DestinationCapabilities
}

type Plan struct {
	ContractVersion    string                   `json:"contractVersion"`
	PlanID             string                   `json:"planId"`
	MetadataRevision   string                   `json:"metadataRevision"`
	MetadataDigest     string                   `json:"metadataDigest"`
	SourceID           string                   `json:"sourceId"`
	ObjectID           string                   `json:"objectId"`
	EffectivePolicy    metadata.EffectivePolicy `json:"effectivePolicy"`
	CapabilityVersions []string                 `json:"capabilityVersions"`
	RecordContract     string                   `json:"recordContract"`
	Stages             []string                 `json:"stages"`
	DeliveryGuarantee  string                   `json:"deliveryGuarantee"`
	Transformations    []string                 `json:"transformations"`
	QualityRules       []string                 `json:"qualityRules"`
	QualityDisposition string                   `json:"qualityDisposition,omitempty"`
}

func PlanRun(document metadata.Document, request Request) (Plan, error) {
	if err := document.Validate(); err != nil {
		return Plan{}, err
	}
	if !document.RunEligible() {
		return Plan{}, domain.NewFailure(domain.FailureInvalidInput, "run.plan", "only active metadata can authorize a plan")
	}
	if !request.Authorized {
		return Plan{}, domain.NewFailure(domain.FailureAccessDenied, "run.plan", "run scope is not authorized")
	}
	if err := ports.ValidateBoundedRun(request.Source, request.Destination); err != nil {
		return Plan{}, err
	}
	policy, err := document.ResolvePolicy(request.SourceID, request.ObjectID)
	if err != nil {
		return Plan{}, err
	}
	if classLevel(request.MaximumClass) < 0 || classLevel(policy.Classification) < 0 || classLevel(policy.Classification) > classLevel(request.MaximumClass) {
		return Plan{}, domain.NewFailure(domain.FailureAccessDenied, "run.plan", "source classification exceeds local policy")
	}
	for _, classification := range policy.ColumnPolicies {
		if classLevel(classification) < 0 || classLevel(classification) > classLevel(request.MaximumClass) {
			return Plan{}, domain.NewFailure(domain.FailureAccessDenied, "run.plan", "column classification exceeds local policy")
		}
	}
	canonical, err := metadata.CanonicalJSON(document)
	if err != nil {
		return Plan{}, domain.NewFailure(domain.FailureInvalidInput, "run.plan", "metadata identity cannot be canonicalized")
	}
	plan := Plan{
		ContractVersion:    RunPlanContractVersion,
		MetadataRevision:   document.Revision,
		MetadataDigest:     metadata.Digest(canonical),
		SourceID:           request.SourceID,
		ObjectID:           request.ObjectID,
		EffectivePolicy:    policy,
		CapabilityVersions: []string{request.Source.Version, request.Destination.Version},
		RecordContract:     RecordContractVersion,
		Stages:             []string{"ingest"},
		DeliveryGuarantee:  "idempotent_bounded_batch_replay",
		Transformations:    make([]string, 0, len(findObject(document, request.SourceID, request.ObjectID).Transformations)),
		QualityRules:       make([]string, 0, len(findObject(document, request.SourceID, request.ObjectID).QualityRules)),
		QualityDisposition: findObject(document, request.SourceID, request.ObjectID).QualityFailureDisposition,
	}
	object := findObject(document, request.SourceID, request.ObjectID)
	for _, rule := range object.Transformations {
		plan.Transformations = append(plan.Transformations, rule.ID+"@"+rule.Version)
	}
	for _, rule := range object.QualityRules {
		plan.QualityRules = append(plan.QualityRules, rule.ID+"@"+rule.Version)
	}
	if len(object.Transformations) > 0 {
		plan.Stages = append(plan.Stages, "transform")
	}
	if len(object.QualityRules) > 0 {
		plan.Stages = append(plan.Stages, "quality", "disposition")
	}
	plan.Stages = append(plan.Stages, "publish")
	identity, err := metadata.CanonicalJSON(struct {
		ContractVersion    string   `json:"contractVersion"`
		MetadataDigest     string   `json:"metadataDigest"`
		SourceID           string   `json:"sourceId"`
		ObjectID           string   `json:"objectId"`
		SourceVersion      string   `json:"sourceVersion"`
		DestinationVersion string   `json:"destinationVersion"`
		RecordContract     string   `json:"recordContract"`
		Guarantee          string   `json:"guarantee"`
		Transformations    []string `json:"transformations"`
		QualityRules       []string `json:"qualityRules"`
		QualityDisposition string   `json:"qualityDisposition,omitempty"`
	}{plan.ContractVersion, plan.MetadataDigest, plan.SourceID, plan.ObjectID, request.Source.Version, request.Destination.Version, plan.RecordContract, plan.DeliveryGuarantee, plan.Transformations, plan.QualityRules, plan.QualityDisposition})
	if err != nil {
		return Plan{}, domain.NewFailure(domain.FailureInternal, "run.plan", "plan identity cannot be created")
	}
	planIdentity, err := domain.NewIdentifier(domain.PlanID, metadata.Digest(identity))
	if err != nil {
		return Plan{}, domain.NewFailure(domain.FailureInternal, "run.plan", "validated plan identity could not be constructed")
	}
	plan.PlanID = planIdentity.String()
	return plan, nil
}

func (p Plan) WithSourceDigest(sourceDigest string) (string, error) {
	if p.ContractVersion != RunPlanContractVersion || p.PlanID == "" || sourceDigest == "" {
		return "", fmt.Errorf("run plan is missing pinned identity")
	}
	if _, err := domain.NewIdentifier(domain.PlanID, p.PlanID); err != nil {
		return "", domain.NewFailure(domain.FailureInvalidInput, "run.plan", "run plan identity is invalid")
	}
	runID, err := domain.NewIdentifier(domain.RunID, metadata.Digest([]byte(p.PlanID+"\x00"+sourceDigest)))
	if err != nil {
		return "", domain.NewFailure(domain.FailureInternal, "run.plan", "run identity could not be constructed")
	}
	return runID.String(), nil
}

func findObject(document metadata.Document, sourceID, objectID string) metadata.Object {
	for _, source := range document.Sources {
		if source.ID != sourceID {
			continue
		}
		for _, object := range source.Objects {
			if object.ID == objectID {
				return object
			}
		}
	}
	return metadata.Object{}
}

func classLevel(value string) int {
	switch value {
	case "public":
		return 0
	case "internal":
		return 1
	case "confidential":
		return 2
	case "restricted":
		return 3
	default:
		return -1
	}
}
