//go:build integration

package local

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalReferenceProfilePersistsAndReconcilesAcrossRunInvocations(t *testing.T) {
	root := t.TempDir()
	stateDir := filepath.Join(root, "state")
	sourceRoot := filepath.Join(root, "source")
	if err := os.MkdirAll(sourceRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	candidate := filepath.Join(root, "metadata.json")
	metadataJSON := `{"schemaVersion":"mimir.metadata/v1","revision":"integration-v1","status":"draft","sources":[{"id":"orders","classification":"internal","objects":[{"id":"orders","path":"orders.jsonl"}]}]}`
	if err := os.WriteFile(candidate, []byte(metadataJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "orders.jsonl"), []byte("{\"id\":1}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	created, err := Author(context.Background(), candidate, stateDir, "local-user", true)
	if err != nil || created.Status != "draft" {
		t.Fatalf("author result=%+v err=%v", created, err)
	}
	activated, err := ActivateDraft(context.Background(), created.Revision, stateDir, "local-user", true, "")
	if err != nil || activated.Status != "active" {
		t.Fatalf("activation result=%+v err=%v", activated, err)
	}

	options := Options{
		SourceRoot: sourceRoot, StateDir: stateDir,
		Actor: "local-user", AllowLocal: true,
	}
	first, err := Run(context.Background(), options)
	if err != nil || first.RunID == "" || first.AlreadyCommitted {
		t.Fatalf("initial run result=%+v err=%v", first, err)
	}
	resumed, err := Run(context.Background(), options)
	if err != nil || resumed.RunID != first.RunID || !resumed.AlreadyCommitted {
		t.Fatalf("reconciled run result=%+v err=%v", resumed, err)
	}
	committed, err := ReadCommitted(filepath.Join(stateDir, "data", "destination.jsonl"))
	if err != nil || len(committed) != 1 || committed[0].BatchID != first.RunID {
		t.Fatalf("durable destination entries=%+v err=%v", committed, err)
	}
	checkpoints, err := readAndRepairTail[checkpoint](filepath.Join(stateDir, "state", "checkpoints.jsonl"))
	if err != nil || len(checkpoints) != 1 || checkpoints[0].RunID != first.RunID {
		t.Fatalf("durable checkpoints=%+v err=%v", checkpoints, err)
	}
}
