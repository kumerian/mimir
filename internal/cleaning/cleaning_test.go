package cleaning

import "testing"

func TestApplyIsExplicitAuditableAndDoesNotReplaceInput(t *testing.T) {
	input := map[string]any{"name": "  Ada "}
	result, err := Apply(input, Correction{
		ID: "trim-name", Version: "v1", Operation: "trim", Field: "name",
	}, "operator-1", true)
	if err != nil {
		t.Fatal(err)
	}
	if result.Record["name"] != "Ada" || input["name"] != "  Ada " {
		t.Fatalf("cleaning did not preserve source: input=%v result=%v", input, result.Record)
	}
	if result.Audit.InputReference == "" || result.Audit.OutputDigest == "" || result.Audit.Actor != "operator-1" {
		t.Fatalf("missing cleaning audit provenance: %+v", result.Audit)
	}
	if _, err := Apply(input, Correction{ID: "trim-name", Version: "v1", Operation: "trim", Field: "name"}, "operator-1", false); err == nil {
		t.Fatal("Apply() accepted unauthorized cleaning")
	}
}

func TestApplyRejectsUnsupportedCorrectionVersion(t *testing.T) {
	_, err := Apply(map[string]any{"name": " Ada "},
		Correction{ID: "trim", Version: "v2", Operation: "trim", Field: "name"},
		"operator-1", true)
	if err == nil {
		t.Fatal("unsupported correction version was accepted")
	}
}
