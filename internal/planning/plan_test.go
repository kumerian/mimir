package planning

import (
	"testing"

	"github.com/kumerian/mimir/internal/metadata"
)

func TestPlanRunRejectsMissingReconciliationCapabilityBeforePlanning(t *testing.T) {
	doc := activeDocument()
	_, err := PlanRun(doc, Request{
		SourceID: "orders", ObjectID: "orders", Authorized: true, MaximumClass: "internal",
		Source:      SourceCapabilities{Version: "local.source/v1", BoundedSnapshot: true, StableIdentity: true},
		Destination: DestinationCapabilities{Version: "local.destination/v1", IdempotentWrites: true, DurableCommitEvidence: true},
	})
	if err == nil {
		t.Fatal("PlanRun() accepted a destination without reconciliation")
	}
}

func TestPlanRunRequiresActiveMetadataAuthorizationAndPinsVersions(t *testing.T) {
	doc := activeDocument()
	request := Request{
		SourceID: "orders", ObjectID: "orders", Authorized: true, MaximumClass: "internal",
		Source:      SourceCapabilities{Version: "local.source/v1", BoundedSnapshot: true, StableIdentity: true},
		Destination: DestinationCapabilities{Version: "local.destination/v1", IdempotentWrites: true, DurableCommitEvidence: true, Reconcile: true},
	}
	plan, err := PlanRun(doc, request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.MetadataRevision != "rev-1" || len(plan.CapabilityVersions) != 2 || plan.DeliveryGuarantee == "" {
		t.Fatalf("plan omitted pinned input or guarantee: %+v", plan)
	}
	doc.Status = "draft"
	if _, err := PlanRun(doc, request); err == nil {
		t.Fatal("PlanRun() accepted draft metadata")
	}
	request.Authorized = false
	doc.Status = "active"
	if _, err := PlanRun(doc, request); err == nil {
		t.Fatal("PlanRun() accepted denied scope")
	}
}

func activeDocument() metadata.Document {
	return metadata.Document{
		SchemaVersion: metadata.ContractVersion, Revision: "rev-1", Status: "active",
		Sources: []metadata.Source{{ID: "orders", Classification: "internal", Objects: []metadata.Object{{ID: "orders", Path: "orders.jsonl"}}}},
	}
}
