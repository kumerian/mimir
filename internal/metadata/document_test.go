package metadata

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseRejectsDuplicateUnknownAndUnsupportedFields(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "duplicate keys", body: `{"schemaVersion":"mimir.metadata/v1","schemaVersion":"mimir.metadata/v1"}`, want: "duplicate"},
		{name: "unknown fields", body: `{"schemaVersion":"mimir.metadata/v1","unexpected":true}`, want: `unknown metadata field "unexpected"`},
		{name: "sensitive-looking unknown field", body: `{"schemaVersion":"mimir.metadata/v1","synthetic-canary":true}`, want: "name withheld"},
		{name: "unsupported version", body: `{"schemaVersion":"mimir.metadata/v99"}`, want: "unsupported"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tt.body))
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), tt.want) {
				t.Fatalf("Parse() error = %v, want text containing %q", err, tt.want)
			}
		})
	}
}

func TestMetadataConstructorsRejectUnsafeRevisionAndEntityIdentifiers(t *testing.T) {
	tests := []string{
		`{"schemaVersion":"mimir.metadata/v1","revision":"../escape","status":"draft","sources":[{"id":"s","classification":"internal","objects":[{"id":"o","path":"o.jsonl"}]}]}`,
		`{"schemaVersion":"mimir.metadata/v1","revision":"valid","status":"draft","sources":[{"id":"not safe","classification":"internal","objects":[{"id":"o","path":"o.jsonl"}]}]}`,
		`{"schemaVersion":"mimir.metadata/v1","revision":"valid","status":"draft","sources":[{"id":"s","classification":"internal","objects":[{"id":"o","path":"o.jsonl","columns":[{"id":"bad/id","type":"string"}]}]}]}`,
	}
	for _, input := range tests {
		if _, err := Parse(strings.NewReader(input)); err == nil {
			t.Fatalf("metadata with unsafe domain identity was accepted: %s", input)
		}
	}
}

func TestParseProducesDraftCandidateFromEverySupportedMetadataEntity(t *testing.T) {
	input := `{"schemaVersion":"mimir.metadata/v1","revision":"complete-draft","status":"draft","sources":[{"id":"orders","classification":"internal","secretReferences":[{"id":"source-token","reference":"env://MIMIR_SOURCE_TOKEN"}],"objects":[{"id":"orders","path":"orders.jsonl","classification":"confidential","columns":[{"id":"order_id","type":"integer","classification":"internal"}],"transformations":[{"id":"trim-name","version":"v1","operation":"trim","field":"customer_name"}],"qualityRules":[{"id":"required-id","version":"v1","operation":"required","field":"order_id"}],"qualityFailureDisposition":"quarantine","schedule":{"enabled":true,"intervalSeconds":60,"retryLimit":2,"concurrencyLimit":1}}]}]}`
	parsed, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() rejected complete valid candidate: %v", err)
	}
	if parsed.Document.SchemaVersion != ContractVersion || parsed.Document.RunEligible() {
		t.Fatalf("candidate version/authority = %q/%v", parsed.Document.SchemaVersion, parsed.Document.RunEligible())
	}
	if len(parsed.Document.Sources) != 1 || len(parsed.Document.Sources[0].SecretRefs) != 1 ||
		len(parsed.Document.Sources[0].Objects[0].Columns) != 1 ||
		len(parsed.Document.Sources[0].Objects[0].Transformations) != 1 ||
		len(parsed.Document.Sources[0].Objects[0].QualityRules) != 1 ||
		parsed.Document.Sources[0].Objects[0].Schedule == nil {
		t.Fatalf("candidate lost supported metadata entities: %+v", parsed.Document)
	}
}

func TestParseValidMetadataIsDeterministic(t *testing.T) {
	input := `{"schemaVersion":"mimir.metadata/v1","revision":"demo-1","status":"active","sources":[{"id":"orders","classification":"internal","objects":[{"id":"orders","path":"orders.jsonl"}]}]}`
	first, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	second, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() second error = %v", err)
	}
	if !bytes.Equal(first.Canonical, second.Canonical) || first.Document.Revision != "demo-1" {
		t.Fatalf("parse is not deterministic: first=%+v second=%+v", first, second)
	}
}

func TestSecretReferencesAreOpaqueAndRestrictivelyValidated(t *testing.T) {
	base := Document{
		SchemaVersion: ContractVersion, Revision: "secret-ref-test", Status: "draft",
		Sources: []Source{{
			ID: "source", Classification: "internal",
			SecretRefs: []SecretReference{{ID: "source-auth", Reference: "env://MIMIR_SOURCE_TOKEN"}},
			Objects:    []Object{{ID: "object", Path: "object.jsonl"}},
		}},
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("valid opaque environment reference rejected: %v", err)
	}
	base.Sources[0].SecretRefs[0].Reference = "file:///tmp/secret"
	if err := base.Validate(); err == nil {
		t.Fatal("unsupported secret reference scheme accepted")
	}
	base.Sources[0].SecretRefs = []SecretReference{
		{ID: "same", Reference: "env://FIRST"},
		{ID: "same", Reference: "env://SECOND"},
	}
	if err := base.Validate(); err == nil {
		t.Fatal("duplicate secret reference identity accepted")
	}
}

func TestMetadataRejectsUnsupportedRuleVersions(t *testing.T) {
	input := `{"schemaVersion":"mimir.metadata/v1","revision":"unsupported-rule","status":"draft","sources":[{"id":"source","classification":"internal","objects":[{"id":"object","path":"object.jsonl","transformations":[{"id":"trim","version":"v2","operation":"trim","field":"name"}]}]}]}`
	if _, err := Parse(strings.NewReader(input)); err == nil {
		t.Fatal("metadata with an unsupported transformation rule version was accepted")
	}
}

func TestResolvePolicyAppliesOnlyMoreRestrictiveOverridesAndTracksOrigin(t *testing.T) {
	doc := Document{Sources: []Source{{
		ID: "source", Classification: "internal",
		Objects: []Object{{
			ID: "object", Classification: "confidential",
			Columns: []Column{
				{ID: "a", Type: "string"},
				{ID: "b", Type: "string", Classification: "restricted"},
			},
		}},
	}}}
	policy, err := doc.ResolvePolicy("source", "object")
	if err != nil {
		t.Fatal(err)
	}
	if policy.Classification != "confidential" || policy.ClassificationAt != "object:object" {
		t.Fatalf("unexpected object policy: %+v", policy)
	}
	if policy.ColumnPolicies["a"] != "confidential" || policy.ColumnOrigins["a"] != "object:object" {
		t.Fatalf("column did not inherit object policy: %+v", policy)
	}
	if policy.ColumnPolicies["b"] != "restricted" || policy.ColumnOrigins["b"] != "column:b" {
		t.Fatalf("column override did not resolve: %+v", policy)
	}
}

func TestCanonicalizeRecordRejectsDuplicateKeysAndNonObjectValues(t *testing.T) {
	if _, err := CanonicalizeRecord([]byte(`{"name":"first","name":"second"}`)); err == nil {
		t.Fatal("duplicate record keys were accepted")
	}
	if _, err := CanonicalizeRecord([]byte(`[1,2]`)); err == nil {
		t.Fatal("non-object record was accepted")
	}
}

func TestScheduleMetadataRejectsUnknownAndCyclicDependencies(t *testing.T) {
	tests := []struct {
		name    string
		objects []Object
	}{
		{
			name: "unknown dependency",
			objects: []Object{{ID: "one", Path: "one.jsonl", Schedule: &Schedule{
				Enabled: true, IntervalSeconds: 60, Dependencies: []string{"missing"},
			}}},
		},
		{
			name: "cycle",
			objects: []Object{
				{ID: "one", Path: "one.jsonl", Schedule: &Schedule{Enabled: true, IntervalSeconds: 60, Dependencies: []string{"two"}}},
				{ID: "two", Path: "two.jsonl", Schedule: &Schedule{Enabled: true, IntervalSeconds: 60, Dependencies: []string{"one"}}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			document := Document{
				SchemaVersion: ContractVersion, Revision: "schedule-test", Status: "active",
				Sources: []Source{{ID: "source", Classification: "internal", Objects: tt.objects}},
			}
			if err := document.Validate(); err == nil {
				t.Fatal("invalid schedule dependency graph was accepted")
			}
		})
	}
}
