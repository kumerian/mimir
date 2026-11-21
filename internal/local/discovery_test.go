package local

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kumerian/mimir/internal/domain"
)

func TestDiscoverEmptyObjectPersistsArrayColumns(t *testing.T) {
	_, sourceDir, stateDir, candidate := setupLocalRun(t, "discover-empty", `{}`)
	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
		t.Fatal(err)
	}

	result, err := Discover(context.Background(), DiscoveryOptions{
		SourceRoot: sourceDir, StateDir: stateDir, Actor: "local-user", AllowLocal: true,
		Disposition: "hold",
	})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	stored, err := os.ReadFile(filepath.Join(stateDir, "metadata", "proposals", result.ProposalID+".json"))
	if err != nil {
		t.Fatalf("reading persisted proposal: %v", err)
	}
	response, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal discovery response: %v", err)
	}
	var responseEnvelope struct {
		Proposal json.RawMessage `json:"proposal"`
	}
	if err := json.Unmarshal(response, &responseEnvelope); err != nil {
		t.Fatalf("discovery response is invalid JSON: %v", err)
	}
	for name, encoded := range map[string][]byte{
		"response": responseEnvelope.Proposal, "stored proposal": stored,
	} {
		var wire struct {
			Observation struct {
				Columns json.RawMessage `json:"columns"`
			} `json:"observation"`
			Proposed json.RawMessage `json:"proposedColumns"`
		}
		if err := json.Unmarshal(encoded, &wire); err != nil {
			t.Fatalf("%s is invalid JSON: %v", name, err)
		}
		if string(wire.Observation.Columns) != "[]" || string(wire.Proposed) != "[]" {
			t.Fatalf("%s columns = %s / %s, want [] / []", name, wire.Observation.Columns, wire.Proposed)
		}
	}
}

func TestDiscoverPersistsVersionedProposalWithoutChangingActiveRevision(t *testing.T) {
	_, sourceDir, stateDir, candidate := setupLocalRun(t, "discover-1", `{"id":1,"name":"Synthetic"}`)
	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
		t.Fatal(err)
	}
	before, pointer, err := loadActiveMetadata(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.OpenFile(filepath.Join(sourceDir, "orders.jsonl"), os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := source.WriteString("{\"id\":2,\"name\":\"Example\",\"total\":12.5}\n"); err != nil {
		t.Fatal(err)
	}
	if err := source.Close(); err != nil {
		t.Fatal(err)
	}
	result, err := Discover(context.Background(), DiscoveryOptions{
		SourceRoot: sourceDir, StateDir: stateDir, Actor: "local-user", AllowLocal: true,
		Disposition: "continue",
	})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if result.Proposal.ExecutionReady || len(result.Proposal.Proposed) != 3 ||
		result.Proposal.Report.Changes[0].Kind != "added" {
		t.Fatalf("unexpected discovery proposal: %+v", result.Proposal)
	}
	after, afterPointer, err := loadActiveMetadata(stateDir)
	if err != nil || afterPointer.MetadataDigest != pointer.MetadataDigest || string(after.Canonical) != string(before.Canonical) {
		t.Fatalf("discovery changed active metadata: err=%v before=%s after=%s", err, before.Canonical, after.Canonical)
	}
	stored, err := os.ReadFile(filepath.Join(stateDir, "metadata", "proposals", result.ProposalID+".json"))
	if err != nil || len(stored) == 0 {
		t.Fatalf("proposal was not persisted: err=%v", err)
	}
	columns, err := inferColumns([][]byte{
		[]byte(`{"id":1,"name":"Synthetic"}`), []byte(`{"id":2,"name":"Example","total":12.5}`),
	})
	if err != nil || len(columns) != 3 || columns[2].Type != "number" {
		t.Fatalf("observation columns=%+v err=%v", columns, err)
	}
	if columns[0].Classification != "" {
		t.Fatalf("discovery unexpectedly assigned classification: %+v", columns[0])
	}
}

func TestDiscoverTimeoutLeavesActiveMetadataAndProposalsUnchanged(t *testing.T) {
	_, sourceDir, stateDir, candidate := setupLocalRun(t, "discover-timeout", `{"id":1}`)
	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
		t.Fatal(err)
	}
	before, beforePointer, err := loadActiveMetadata(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	_, err = Discover(ctx, DiscoveryOptions{
		SourceRoot: sourceDir, StateDir: stateDir, Actor: "local-user", AllowLocal: true,
		Disposition: "hold",
	})
	var failure domain.Failure
	if !errors.As(err, &failure) || failure.Category != domain.FailureTimeout {
		t.Fatalf("timed-out discovery=%#v, want timeout failure", err)
	}
	after, afterPointer, err := loadActiveMetadata(stateDir)
	if err != nil || afterPointer.MetadataDigest != beforePointer.MetadataDigest ||
		string(after.Canonical) != string(before.Canonical) {
		t.Fatalf("timed-out discovery changed active metadata: err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(stateDir, "metadata", "proposals")); !os.IsNotExist(err) {
		t.Fatalf("timed-out discovery persisted a proposal: %v", err)
	}
}

func TestInferColumnsRejectsInconsistentAndUnobservableTypes(t *testing.T) {
	if _, err := inferColumns([][]byte{[]byte(`{"id":1}`), []byte(`{"id":"one"}`)}); err == nil {
		t.Fatal("inconsistent observed types were accepted")
	}
	if _, err := inferColumns([][]byte{[]byte(`{"value":null}`)}); err == nil {
		t.Fatal("unobservable null-only field was assigned a guessed type")
	}
}
