package local

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kumerian/mimir/internal/domain"

	"github.com/kumerian/mimir/internal/contracttest"
)

func TestLocalReferenceAdaptersSatisfySharedContracts(t *testing.T) {
	root := t.TempDir()
	sourceRoot := filepath.Join(root, "source")
	if err := os.Mkdir(sourceRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "records.jsonl"), []byte("{\"id\":1}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	contracttest.RunSourceContract(t, fileSourceAdapter{root: sourceRoot}, "records.jsonl")
	destinationPath := filepath.Join(root, "data", "destination.jsonl")
	if err := ensurePrivateDirectory(filepath.Dir(destinationPath)); err != nil {
		t.Fatal(err)
	}
	contracttest.RunDestinationContract(t, fileDestinationAdapter{path: destinationPath})
}

func TestReferenceAdaptersClassifyDeadlinesAndUnsupportedAcknowledgement(t *testing.T) {
	root := t.TempDir()
	sourceRoot := filepath.Join(root, "source")
	if err := os.Mkdir(sourceRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "records.jsonl"), []byte("{\"id\":1}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	expired, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	_, _, err := (fileSourceAdapter{root: sourceRoot}).ReadBatch(expired, "records.jsonl")
	failure, ok := err.(domain.Failure)
	if !ok || failure.Category != domain.FailureTimeout {
		t.Fatalf("source deadline failure=%#v, want timeout", err)
	}
	if err := (fileSourceAdapter{root: sourceRoot}).Acknowledge(context.Background(), "position"); err == nil {
		t.Fatal("source accepted unsupported acknowledgement")
	}
	destination := fileDestinationAdapter{path: filepath.Join(root, "data", "destination.jsonl")}
	_, err = destination.Reconcile(expired, "batch", nil)
	failure, ok = err.(domain.Failure)
	if !ok || failure.Category != domain.FailureTimeout {
		t.Fatalf("destination deadline failure=%#v, want timeout", err)
	}
	if _, err := os.Stat(destination.path); !os.IsNotExist(err) {
		t.Fatalf("expired destination operation caused a side effect: %v", err)
	}
}

func TestReferenceAdaptersClassifyCancellationDeniedScopeAndUnavailableSource(t *testing.T) {
	root := t.TempDir()
	sourceRoot := filepath.Join(root, "source")
	if err := os.Mkdir(sourceRoot, 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(sourceRoot, "records.jsonl"), []byte("{\"id\":1}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "outside.jsonl"), []byte("{\"id\":2}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	source := fileSourceAdapter{root: sourceRoot}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := source.ReadBatch(ctx, "records.jsonl")
	failure, ok := err.(domain.Failure)
	if !ok || failure.Category != domain.FailureCancelled {
		t.Fatalf("source cancellation failure=%#v", err)
	}
	_, _, err = source.ReadBatch(context.Background(), "../outside.jsonl")
	failure, ok = err.(domain.Failure)
	if !ok || failure.Category != domain.FailureAccessDenied {
		t.Fatalf("denied source scope failure=%#v", err)
	}
	_, _, err = source.ReadBatch(context.Background(), "missing.jsonl")
	failure, ok = err.(domain.Failure)
	if !ok || failure.Category != domain.FailureUnavailable || failure.CauseReference == "" {
		t.Fatalf("unavailable source failure=%#v, want stable failure with cause reference", err)
	}
	if strings.Contains(failure.Error(), "missing.jsonl") || strings.Contains(failure.Error(), sourceRoot) {
		t.Fatalf("source filesystem cause leaked through Mimir failure: %v", failure)
	}
}

func TestDestinationAdapterMapsTechnologyCauseToStableFailureReference(t *testing.T) {
	destination := fileDestinationAdapter{path: filepath.Join(t.TempDir(), "missing-parent", "destination.jsonl")}
	payload, err := json.Marshal(committedBatch{
		BatchID: "batch-1", CommitEvidence: "commit-1", Records: []json.RawMessage{},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = destination.CommitBatch(context.Background(), "batch-1", payload)
	failure, ok := err.(domain.Failure)
	if !ok || failure.Category != domain.FailureUnavailable ||
		failure.CauseReference == "" || failure.CommitCertainty != domain.CommitNotCommitted {
		t.Fatalf("destination adapter failure=%#v, want stable unavailable failure with cause reference", err)
	}
	if strings.Contains(failure.Error(), "missing-parent") {
		t.Fatalf("adapter filesystem error leaked through Mimir failure: %v", failure)
	}
}
