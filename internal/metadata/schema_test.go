package metadata

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEmbeddedMetadataSchemaLoadsAndValidates(t *testing.T) {
	schema, err := loadV1Schema()
	if err != nil {
		t.Fatal(err)
	}
	if schema == nil {
		t.Fatal("embedded metadata schema compiled to nil")
	}
}

func TestValidateSchemaRejectsInvalidMetadataDeterministically(t *testing.T) {
	valid := []byte(`{"schemaVersion":"mimir.metadata/v1","revision":"schema-test","status":"draft","sources":[{"id":"source","classification":"internal","objects":[{"id":"object","path":"object.jsonl"}]}]}`)
	if diagnostics := ValidateSchema(valid); len(diagnostics) != 0 {
		t.Fatalf("valid document diagnostics = %+v", diagnostics)
	}

	invalid := []byte(`{"schemaVersion":"mimir.metadata/v1","revision":"schema-test","status":"unknown","sources":[{"id":"source","classification":"internal","objects":[{"id":"object","path":"/private/path"}]}]}`)
	first := ValidateSchema(invalid)
	second := ValidateSchema(invalid)
	if len(first) == 0 {
		t.Fatal("schema accepted invalid status and absolute path")
	}
	if len(first) != len(second) {
		t.Fatalf("diagnostic count differs across runs: %d != %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("diagnostics are not deterministic: first=%+v second=%+v", first, second)
		}
		if strings.Contains(first[i].Message, "private") || strings.Contains(first[i].Message, "/private/path") {
			t.Fatalf("diagnostic disclosed metadata content: %+v", first[i])
		}
	}
}

func TestValidateSchemaRejectsUnknownFieldsAndUnsupportedVersions(t *testing.T) {
	for _, input := range []string{
		`{"schemaVersion":"mimir.metadata/v1","revision":"schema-test","status":"draft","sources":[],"unknown":true}`,
		`{"schemaVersion":"mimir.metadata/v99","revision":"schema-test","status":"draft","sources":[]}`,
	} {
		if diagnostics := ValidateSchema([]byte(input)); len(diagnostics) == 0 {
			t.Fatalf("schema accepted invalid metadata: %s", input)
		}
	}
}

func TestValidateSchemaRejectsUnsafeObjectPathsAndRedactsSensitiveFieldNames(t *testing.T) {
	unsafePaths := []string{"/absolute/file.jsonl", "../escape.jsonl", "folder\\file.jsonl", "drive:C/file"}
	for _, path := range unsafePaths {
		encodedPath, err := json.Marshal(path)
		if err != nil {
			t.Fatal(err)
		}
		input := `{"schemaVersion":"mimir.metadata/v1","revision":"schema-path","status":"draft","sources":[{"id":"s","classification":"internal","objects":[{"id":"o","path":` +
			string(encodedPath) + `}]}]}`
		if diagnostics := ValidateSchema([]byte(input)); len(diagnostics) == 0 {
			t.Fatalf("schema accepted unsafe path %q", path)
		}
	}

	canary := "synthetic-secret-canary"
	input := []byte(`{"schemaVersion":"mimir.metadata/v1","revision":"schema-secret","status":"draft","sources":[{"id":"s","classification":"internal","objects":[{"id":"o","path":"o.jsonl"}]}],"` + canary + `":"private-value"}`)
	diagnostics := ValidateSchema(input)
	if len(diagnostics) == 0 {
		t.Fatal("schema accepted an unknown sensitive-looking field")
	}
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Path, canary) || strings.Contains(diagnostic.Message, canary) ||
			strings.Contains(diagnostic.Message, "private-value") {
			t.Fatalf("schema diagnostic leaked sensitive input: %+v", diagnostic)
		}
	}
}

func TestValidationReportIncludesDeterministicCompatibilityAndSecurityDecisions(t *testing.T) {
	valid := Validate([]byte(`{"schemaVersion":"mimir.metadata/v1","revision":"report","status":"draft","sources":[{"id":"s","classification":"internal","objects":[{"id":"o","path":"o.jsonl"}]}]}`))
	if !SafeValidationReport(valid) || valid.CompatibilityDecision != "accepted_v1" ||
		valid.SecurityImpact != "input_not_retained" || valid.RunEligible {
		t.Fatalf("valid report has incorrect decisions: %+v", valid)
	}
	unsupported := Validate([]byte(`{"schemaVersion":"mimir.metadata/v99"}`))
	if !SafeValidationReport(unsupported) || unsupported.CompatibilityDecision != "unsupported_version" ||
		unsupported.Valid || unsupported.RunEligible {
		t.Fatalf("unsupported report has incorrect decisions: %+v", unsupported)
	}
}
