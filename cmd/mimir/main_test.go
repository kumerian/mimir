package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/kumerian/mimir/internal/local"
)

func TestHealthReportsBuildAndContractIdentity(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := execute([]string{"health"}, &stdout, &stderr); code != 0 {
		t.Fatalf("health exit code = %d, stderr=%s", code, stderr.String())
	}
	var result health
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("health output is not valid JSON: %v", err)
	}
	if !sort.StringsAreSorted(result.Contracts) {
		t.Fatalf("health contracts are not deterministic: %v", result.Contracts)
	}
	seen := make(map[string]bool, len(result.Contracts))
	for _, contract := range result.Contracts {
		if seen[contract] {
			t.Fatalf("health reported duplicate contract %q", contract)
		}
		seen[contract] = true
	}
	for _, expected := range []string{
		`"status":"ready"`, `"mimir.metadata/v1"`, `"local-reference-v1"`,
		`"mimir.run-plan/v1"`, `"mimir.run-state/v1"`, `"mimir.discovery/v1"`,
		`"local.source/v1"`, `"local.destination/v1"`, `"mimir.secret-resolver/v1"`,
		`"mimir.run-state-store/v1"`, `"mimir.source-adapter/v1"`,
		`"mimir.destination-adapter/v1"`, `"mimir.record/json-object/v1"`,
		`"mimir.replay/v1"`, `"mimir.audit-store/v1"`, `"mimir.telemetry/v1"`,
	} {
		if !strings.Contains(stdout.String(), expected) {
			t.Errorf("health output %q does not contain %q", stdout.String(), expected)
		}
	}
}

func TestValidateCommandPersistsAndQueriesNonEligibleReport(t *testing.T) {
	root := t.TempDir()
	stateDir := filepath.Join(root, "state")
	metadataPath := filepath.Join(root, "metadata.json")
	raw := `{"schemaVersion":"mimir.metadata/v1","revision":"cli-report","status":"draft","sources":[{"id":"source","classification":"internal","objects":[{"id":"object","path":"object.jsonl"}]}]}`
	if err := os.WriteFile(metadataPath, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := execute([]string{"validate", "--metadata", metadataPath, "--state-dir", stateDir}, &stdout, &stderr); code != 0 {
		t.Fatalf("validate exit=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var report struct {
		InputDigest string `json:"inputDigest"`
		Valid       bool   `json:"valid"`
		RunEligible bool   `json:"runEligible"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil || !report.Valid || report.RunEligible || report.InputDigest == "" {
		t.Fatalf("validation result=%+v err=%v", report, err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := execute([]string{"validation-report", "--state-dir", stateDir, "--input-digest", report.InputDigest}, &stdout, &stderr); code != 0 {
		t.Fatalf("validation-report exit=%d stderr=%s", code, stderr.String())
	}
	var loaded struct {
		InputDigest string `json:"inputDigest"`
		Valid       bool   `json:"valid"`
		RunEligible bool   `json:"runEligible"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &loaded); err != nil || loaded != report {
		t.Fatalf("queried validation report=%+v, expected=%+v err=%v", loaded, report, err)
	}
}

func TestRunRequiresExplicitLocalGrant(t *testing.T) {
	var stdout, stderr bytes.Buffer
	stateDir := filepath.Join(t.TempDir(), "state")
	code := execute([]string{"run", "--source-root", "not-read", "--state-dir", stateDir}, &stdout, &stderr)
	if code != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "authorization") {
		t.Fatalf("run result: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	audit, err := os.ReadFile(filepath.Join(stateDir, "audit", "events.jsonl"))
	if err != nil || !strings.Contains(string(audit), `"operation":"access.denied"`) {
		t.Fatalf("denied run did not record redacted denial evidence: %v", err)
	}
}

func TestLocalActivateAndRunCommandsDoNotEmitRecordValues(t *testing.T) {
	root := t.TempDir()
	stateDir := filepath.Join(root, "state")
	sourceDir := filepath.Join(root, "source")
	if err := os.MkdirAll(sourceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	metadataPath := filepath.Join(root, "metadata.json")
	metadata := `{"schemaVersion":"mimir.metadata/v1","revision":"cli-1","status":"draft","sources":[{"id":"orders","classification":"internal","objects":[{"id":"orders","path":"orders.jsonl"}]}]}`
	if err := os.WriteFile(metadataPath, []byte(metadata), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "orders.jsonl"), []byte("{\"id\":9,\"canary\":\"synthetic-only-value\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := execute([]string{"activate", "--metadata", metadataPath, "--state-dir", stateDir, "--allow-local"}, &stdout, &stderr); code != 0 {
		t.Fatalf("activate command: code=%d stderr=%s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := execute([]string{"run", "--source-root", sourceDir, "--state-dir", stateDir, "--allow-local"}, &stdout, &stderr); code != 0 {
		t.Fatalf("run command: code=%d stderr=%s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "synthetic-only-value") || strings.Contains(stderr.String(), "synthetic-only-value") {
		t.Fatal("command output leaked a record value")
	}
	committed, err := local.ReadCommitted(filepath.Join(stateDir, "data", "destination.jsonl"))
	if err != nil || len(committed) != 1 {
		t.Fatalf("destination result: batches=%d err=%v", len(committed), err)
	}
	audit, err := os.ReadFile(filepath.Join(stateDir, "audit", "events.jsonl"))
	if err != nil || strings.Contains(string(audit), "synthetic-only-value") {
		t.Fatalf("audit contained record data or was unavailable: %v", err)
	}
}
