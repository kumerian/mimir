package discovery

import (
	"reflect"
	"testing"

	"github.com/kumerian/mimir/internal/metadata"
)

func TestClassifyProducesStableProposalWithoutChangingActiveMetadata(t *testing.T) {
	active := []metadata.Column{{ID: "id", Type: "integer"}, {ID: "name", Type: "string"}}
	observed := []metadata.Column{{ID: "id", Type: "string"}, {ID: "email", Type: "string"}}
	before := append([]metadata.Column(nil), active...)
	report, err := Classify(active, observed, "hold")
	if err != nil {
		t.Fatal(err)
	}
	if report.Disposition != "hold" || report.Compatible {
		t.Fatalf("unexpected drift report: %+v", report)
	}
	if !reflect.DeepEqual(active, before) {
		t.Fatalf("classification mutated active metadata: %+v", active)
	}
	if len(report.Changes) != 3 || report.Changes[0].Field != "email" {
		t.Fatalf("changes are not normalized and sorted: %+v", report.Changes)
	}
}

func TestClassifyRequiresExplicitPolicyAndDoesNotContinueIncompatibleChanges(t *testing.T) {
	active := []metadata.Column{{ID: "id", Type: "integer"}}
	additive := []metadata.Column{{ID: "id", Type: "integer"}, {ID: "name", Type: "string"}}
	if _, err := Classify(active, additive, ""); err == nil {
		t.Fatal("Classify() accepted missing drift policy")
	}

	if _, err := Classify(active, []metadata.Column{{ID: "id", Type: "string"}}, "continue"); err == nil {
		t.Fatal("Classify() silently continued after an incompatible type change")
	}
}

func TestClassifyIsDeterministicAcrossConfiguredDispositions(t *testing.T) {
	active := []metadata.Column{{ID: "id", Type: "integer"}}
	observed := []metadata.Column{{ID: "id", Type: "integer"}, {ID: "name", Type: "string"}}
	for _, disposition := range []string{"reject", "hold", "quarantine", "continue"} {
		first, err := Classify(active, observed, disposition)
		if err != nil {
			t.Fatalf("Classify(%q): %v", disposition, err)
		}
		second, err := Classify(active, observed, disposition)
		if err != nil || !reflect.DeepEqual(first, second) {
			t.Fatalf("Classify(%q) was nondeterministic: first=%+v second=%+v err=%v", disposition, first, second, err)
		}
		if first.Disposition != disposition || !first.Compatible || len(first.Changes) != 1 ||
			first.Changes[0].Field != "name" || first.Changes[0].Kind != "added" {
			t.Fatalf("Classify(%q) returned unexpected disposition: %+v", disposition, first)
		}
	}
}
