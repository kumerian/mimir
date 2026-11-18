package transform

import (
	"reflect"
	"testing"
)

func TestApplyUsesPinnedOrderedRulesWithoutMutatingInput(t *testing.T) {
	input := map[string]any{"name": "  MIMIR ", "legacy_id": "42"}
	rules := []Rule{
		{ID: "trim-name", Version: "v1", Operation: "trim", Field: "name"},
		{ID: "lower-name", Version: "v1", Operation: "lowercase", Field: "name"},
		{ID: "rename-id", Version: "v1", Operation: "rename", Field: "legacy_id", TargetField: "id"},
		{ID: "parse-id", Version: "v1", Operation: "to_integer", Field: "id"},
	}
	result, err := Apply(input, rules)
	if err != nil {
		t.Fatal(err)
	}
	if result.Record["name"] != "mimir" || result.Record["id"] != int64(42) {
		t.Fatalf("unexpected transformed record: %#v", result.Record)
	}
	if !reflect.DeepEqual(input, map[string]any{"name": "  MIMIR ", "legacy_id": "42"}) {
		t.Fatalf("input was mutated: %#v", input)
	}
	if len(result.Provenance) != len(rules) || result.InputDigest == result.OutputDigest {
		t.Fatalf("missing transformation provenance: %+v", result)
	}
}

func TestApplyRejectsUnknownOrUnversionedRule(t *testing.T) {
	for _, rule := range []Rule{
		{ID: "x", Operation: "trim", Field: "name"},
		{ID: "x", Version: "v1", Operation: "execute", Field: "name"},
	} {
		if _, err := Apply(map[string]any{"name": "x"}, []Rule{rule}); err == nil {
			t.Errorf("Apply() accepted unsupported rule: %+v", rule)
		}
	}
}

func TestApplyRejectsUnsupportedRuleVersion(t *testing.T) {
	_, err := Apply(map[string]any{"name": "Ada"}, []Rule{
		{ID: "trim", Version: "v2", Operation: "trim", Field: "name"},
	})
	if err == nil {
		t.Fatal("unsupported transformation rule version was accepted")
	}
}
