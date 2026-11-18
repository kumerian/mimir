package local

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/kumerian/mimir/internal/metadata"
	"github.com/kumerian/mimir/internal/ports"
)

func TestFileMetadataRepositoryPersistsDraftAndConditionallyActivates(t *testing.T) {
	repository := fileMetadataRepository{stateDir: filepath.Join(t.TempDir(), "state")}
	ctx := context.Background()
	raw := []byte(draftMetadata("repository-1"))
	if err := repository.AppendDraft(ctx, raw, "local-user"); err != nil {
		t.Fatal(err)
	}
	if err := repository.AppendDraft(ctx, raw, "local-user"); err != nil {
		t.Fatalf("idempotent draft append: %v", err)
	}
	revisions, err := repository.List(ctx)
	if err != nil || len(revisions) != 1 {
		t.Fatalf("draft list length=%d err=%v", len(revisions), err)
	}
	if _, err := repository.LoadActive(ctx); err == nil {
		t.Fatal("repository exposed draft as active metadata")
	}
	active := []byte(strings.Replace(string(raw), `"status":"draft"`, `"status":"active"`, 1))
	if _, err := repository.Activate(ctx, active, ""); err != nil {
		t.Fatalf("initial activation: %v", err)
	}
	loaded, err := repository.LoadActive(ctx)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := metadata.Parse(strings.NewReader(string(loaded)))
	if err != nil || parsed.Document.Status != "active" {
		t.Fatalf("loaded active metadata=%s err=%v", loaded, err)
	}
	if _, err := repository.Activate(ctx, []byte(strings.Replace(string(draftMetadata("repository-2")), `"status":"draft"`, `"status":"active"`, 1)), "stale"); err == nil {
		t.Fatal("repository accepted stale expected-active precondition")
	}
	unchanged, err := repository.LoadActive(ctx)
	if err != nil || string(unchanged) != string(loaded) {
		t.Fatalf("stale activation changed active metadata: err=%v", err)
	}
	if repository.Capabilities().ContractVersion != ports.MetadataRepositoryVersion {
		t.Fatal("metadata repository did not declare its version")
	}
}

func TestFileMetadataRepositoryAllowsOnlyOneConcurrentInitialActivation(t *testing.T) {
	repository := fileMetadataRepository{stateDir: filepath.Join(t.TempDir(), "state")}
	candidates := [][]byte{
		[]byte(strings.Replace(string(draftMetadata("race-a")), `"status":"draft"`, `"status":"active"`, 1)),
		[]byte(strings.Replace(string(draftMetadata("race-b")), `"status":"draft"`, `"status":"active"`, 1)),
	}
	var wait sync.WaitGroup
	errors := make(chan error, len(candidates))
	wait.Add(len(candidates))
	for _, candidate := range candidates {
		go func(candidate []byte) {
			defer wait.Done()
			_, err := repository.Activate(context.Background(), candidate, "")
			errors <- err
		}(candidate)
	}
	wait.Wait()
	close(errors)
	successes := 0
	for err := range errors {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent activation successes=%d, want one", successes)
	}
	active, err := repository.LoadActive(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(active, &document); err != nil || document["status"] != "active" {
		t.Fatalf("active metadata was not durable: %s err=%v", active, err)
	}
}

func TestFileMetadataRepositoryRejectsUnsupportedVersionBeforeSideEffects(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	repository := fileMetadataRepository{stateDir: stateDir}
	if err := repository.AppendDraft(context.Background(), []byte(`{"schemaVersion":"mimir.metadata/v99"}`), "local-user"); err == nil {
		t.Fatal("repository accepted unsupported metadata version")
	}
	if _, err := repository.List(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.LoadActive(context.Background()); err == nil {
		t.Fatal("unsupported metadata caused an active revision")
	}
}
