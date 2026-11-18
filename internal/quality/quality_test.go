package quality

import "testing"

func TestEvaluateReportsDispositionWithoutMutatingRecord(t *testing.T) {
	record := map[string]any{"name": "", "age": 17}
	result, err := Evaluate(record, []Rule{
		{ID: "name-required", Version: "v1", Operation: "required", Field: "name"},
		{ID: "age-min", Version: "v1", Operation: "min", Field: "age", Number: 18},
	}, "quarantine")
	if err != nil {
		t.Fatal(err)
	}
	if result.Passed || result.Disposition != "quarantine" || len(result.Failures) != 2 {
		t.Fatalf("unexpected quality result: %+v", result)
	}
	if record["name"] != "" || record["age"] != 17 {
		t.Fatalf("quality evaluation mutated input: %#v", record)
	}
}

func TestEvaluateRequiresExplicitFailureDisposition(t *testing.T) {
	if _, err := Evaluate(map[string]any{"x": ""}, []Rule{{ID: "required", Version: "v1", Operation: "required", Field: "x"}}, ""); err == nil {
		t.Fatal("Evaluate() accepted missing disposition policy")
	}
}

func TestEvaluateRejectsUnsupportedRuleVersion(t *testing.T) {
	_, err := Evaluate(map[string]any{"name": "Ada"}, []Rule{
		{ID: "required", Version: "v2", Operation: "required", Field: "name"},
	}, "reject")
	if err == nil {
		t.Fatal("unsupported quality rule version was accepted")
	}
}
