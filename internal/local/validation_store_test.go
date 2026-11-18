package local

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kumerian/mimir/internal/metadata"
)

func TestMetadataValidationReportsAreDeterministicImmutableAndNeverEligible(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	ctx := context.Background()
	raw := []byte(`{"schemaVersion":"mimir.metadata/v1","revision":"report-1","status":"draft","sources":[{"id":"source","classification":"internal","objects":[{"id":"object","path":"object.jsonl"}]}]}`)

	first, err := ValidateMetadata(ctx, raw, stateDir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ValidateMetadata(ctx, raw, stateDir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) || !first.Valid || first.RunEligible {
		t.Fatalf("validation report is not deterministic or safe: first=%+v second=%+v", first, second)
	}
	loaded, err := LoadMetadataValidationReport(ctx, stateDir, first.InputDigest)
	if err != nil || !reflect.DeepEqual(first, loaded) {
		t.Fatalf("loaded validation report=%+v err=%v", loaded, err)
	}

	invalidRaw := []byte(`{"schemaVersion":"mimir.metadata/v1","revision":"report-2","status":"draft","sources":[{"id":"source","classification":"internal","objects":[]}]}`)
	invalid, err := ValidateMetadata(ctx, invalidRaw, stateDir)
	if err != nil {
		t.Fatal(err)
	}
	if invalid.Valid || invalid.RunEligible || len(invalid.Diagnostics) == 0 {
		t.Fatalf("invalid report conferred eligibility or omitted diagnostics: %+v", invalid)
	}
	if _, err := (fileMetadataRepository{stateDir: stateDir}).LoadActive(ctx); err == nil {
		t.Fatal("validation report made invalid metadata active")
	}

	reportPath := filepath.Join(stateDir, "metadata", "validation-reports", first.InputDigest+".json")
	before, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	first.Valid = false
	first.Diagnostics = []metadata.SchemaDiagnostic{{Code: "changed", Path: "/", Message: "changed"}}
	if err := (fileMetadataValidationStore{stateDir: stateDir}).Append(ctx, first); err == nil {
		t.Fatal("validation report mutation replaced the immutable original")
	}
	after, err := os.ReadFile(reportPath)
	if err != nil || string(before) != string(after) {
		t.Fatalf("validation report changed: err=%v", err)
	}
}

func TestMetadataValidationReportRejectsEligibilityAndMalformedContract(t *testing.T) {
	store := fileMetadataValidationStore{stateDir: filepath.Join(t.TempDir(), "state")}
	report := metadata.Validate([]byte(`{"schemaVersion":"mimir.metadata/v99"}`))
	report.RunEligible = true
	if err := store.Append(context.Background(), report); err == nil {
		t.Fatal("store accepted a report with execution authority")
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 {
		t.Fatal("report encoding unexpectedly empty")
	}
}

func TestAuthoringInvalidMetadataPersistsReportWithoutCreatingRevision(t *testing.T) {
	root := t.TempDir()
	stateDir := filepath.Join(root, "state")
	candidate := filepath.Join(root, "invalid.json")
	raw := []byte(`{"schemaVersion":"mimir.metadata/v1","revision":"invalid-draft","status":"draft","sources":[]}`)
	if err := os.WriteFile(candidate, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	_, authorErr := Author(context.Background(), candidate, stateDir, "local-user", true)
	if authorErr == nil {
		t.Fatal("authoring accepted invalid metadata")
	}
	digest := metadata.Validate(raw).InputDigest
	if !strings.Contains(authorErr.Error(), digest) || strings.Contains(authorErr.Error(), string(raw)) {
		t.Fatalf("authoring failure did not provide a safe report identity: %v", authorErr)
	}
	report, err := LoadMetadataValidationReport(context.Background(), stateDir, digest)
	if err != nil || report.Valid || report.RunEligible || len(report.Diagnostics) == 0 {
		t.Fatalf("invalid authoring report=%+v err=%v", report, err)
	}
	revisions, err := (fileMetadataRepository{stateDir: stateDir}).List(context.Background())
	if err != nil || len(revisions) != 0 {
		t.Fatalf("invalid authoring left revisions=%d err=%v", len(revisions), err)
	}
}
