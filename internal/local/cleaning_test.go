package local

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kumerian/mimir/internal/cleaning"
)

func TestCleanRecordPreviewsAndPersistsExplicitAuditedOutput(t *testing.T) {
	root := t.TempDir()
	recordPath := filepath.Join(root, "record.json")
	if err := os.WriteFile(recordPath, []byte(`{"name":" Example "}`), 0o600); err != nil {
		t.Fatal(err)
	}

	stateDir := filepath.Join(root, "state")
	options := CleaningOptions{
		StateDir: stateDir, RecordPath: recordPath, Actor: "local-user",
		Classification: "internal", AllowLocal: true,
		Correction: cleaning.Correction{ID: "trim-name", Version: "v1", Operation: "trim", Field: "name"},
	}
	options.Preview = true
	preview, err := CleanRecord(context.Background(), options)
	if err != nil || !preview.Preview || preview.Record["name"] != "Example" {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
	if _, err := os.Stat(stateDir); !os.IsNotExist(err) {
		t.Fatalf("preview created state: %v", err)
	}
	options.Preview = false
	first, err := CleanRecord(context.Background(), options)
	if err != nil || first.Preview || first.Record["name"] != "Example" || first.AlreadyApplied {
		t.Fatalf("first apply=%+v err=%v", first, err)
	}
	second, err := CleanRecord(context.Background(), options)
	if err != nil || !second.AlreadyApplied || second.Audit.IdempotencyKey != first.Audit.IdempotencyKey {
		t.Fatalf("idempotent apply=%+v err=%v", second, err)
	}
	entries, err := readAndRepairTail[cleanedEntry](filepath.Join(stateDir, "data", "cleaned.jsonl"))
	if err != nil || len(entries) != 1 || string(entries[0].Record) != `{"name":"Example"}` {
		t.Fatalf("cleaned entries=%+v err=%v", entries, err)
	}
	audit, err := os.ReadFile(filepath.Join(stateDir, "audit", "events.jsonl"))
	if err != nil || strings.Contains(string(audit), "Example") ||
		!strings.Contains(string(audit), `"operation":"cleaning.applied"`) {
		t.Fatalf("cleaning audit leaked a record or was absent: err=%v audit=%s", err, audit)
	}
	original, err := os.ReadFile(recordPath)
	if err != nil || string(original) != `{"name":" Example "}` {
		t.Fatalf("cleaning modified original input: %s err=%v", original, err)
	}
}

func TestCleanRecordReconcilesPersistedOutputWhenAuditAppendWasInterrupted(t *testing.T) {
	root := t.TempDir()
	stateDir := filepath.Join(root, "state")
	if err := os.MkdirAll(filepath.Join(stateDir, "audit"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "audit", "events.jsonl"), []byte("{malformed}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	recordPath := filepath.Join(root, "record.json")
	if err := os.WriteFile(recordPath, []byte(`{"name":" Example "}`), 0o600); err != nil {
		t.Fatal(err)
	}
	options := CleaningOptions{
		StateDir: stateDir, RecordPath: recordPath, Actor: "local-user",
		Classification: "internal", AllowLocal: true,
		Correction: cleaning.Correction{ID: "trim-name", Version: "v1", Operation: "trim", Field: "name"},
	}
	if _, err := CleanRecord(context.Background(), options); err == nil {
		t.Fatal("cleaning reported success without required audit evidence")
	}
	outputPath := filepath.Join(stateDir, "data", "cleaned.jsonl")
	if entries, err := readAndRepairTail[cleanedEntry](outputPath); err != nil || len(entries) != 1 {
		t.Fatalf("durable cleaning output missing after interrupted audit: entries=%+v err=%v", entries, err)
	}
	if err := os.Remove(filepath.Join(stateDir, "audit", "events.jsonl")); err != nil {
		t.Fatal(err)
	}
	recovered, err := CleanRecord(context.Background(), options)
	if err != nil || !recovered.AlreadyApplied {
		t.Fatalf("retry failed to reconcile existing output and append audit: result=%+v err=%v", recovered, err)
	}
	audit, err := os.ReadFile(filepath.Join(stateDir, "audit", "events.jsonl"))
	if err != nil || !strings.Contains(string(audit), `"operation":"cleaning.applied"`) {
		t.Fatalf("recovered audit event missing: audit=%s err=%v", audit, err)
	}
}
