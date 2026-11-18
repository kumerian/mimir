// Package contracttest provides reusable behavioral checks for adapter consumers.
package contracttest

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/kumerian/mimir/internal/ports"
)

func RunSourceContract(t testing.TB, adapter ports.SourceAdapter, scope string) {
	t.Helper()
	capabilities := adapter.Capabilities()
	if capabilities.Version == "" || !capabilities.BoundedSnapshot || !capabilities.StableIdentity {
		t.Fatal("source must declare versioned bounded-snapshot and stable-identity capabilities")
	}
	first, firstPosition, err := adapter.ReadBatch(context.Background(), scope)
	if err != nil {
		t.Fatalf("first ReadBatch() error: %v", err)
	}
	second, secondPosition, err := adapter.ReadBatch(context.Background(), scope)
	if err != nil {
		t.Fatalf("second ReadBatch() error: %v", err)
	}
	if firstPosition == "" || firstPosition != secondPosition || len(first) != len(second) {
		t.Fatalf("source identity/position is unstable: first position=%q second=%q", firstPosition, secondPosition)
	}
	for i := range first {
		if !bytes.Equal(first[i], second[i]) {
			t.Fatalf("source record %d changed between identical snapshot reads", i)
		}
	}
	if capabilities.Acknowledgement {
		t.Fatal("local bounded snapshot contract requires no acknowledgement capability")
	}
}

func RunDestinationContract(t testing.TB, adapter ports.DestinationAdapter) {
	t.Helper()
	capabilities := adapter.Capabilities()
	if capabilities.Version == "" || !capabilities.IdempotentWrites ||
		!capabilities.DurableCommitEvidence || !capabilities.Reconcile {
		t.Fatal("destination lacks required idempotency, durable evidence, or reconciliation capability")
	}
	payload := json.RawMessage(`{"batchId":"contract-batch","metadataRevision":"rev","sourceDigest":"source","records":[],"commitEvidence":"evidence"}`)
	committed, err := adapter.Reconcile(context.Background(), "contract-batch", payload)
	if err != nil || committed {
		t.Fatalf("initial Reconcile() committed=%v err=%v, want not committed", committed, err)
	}
	evidence, err := adapter.CommitBatch(context.Background(), "contract-batch", payload)
	if err != nil || evidence != "evidence" {
		t.Fatalf("CommitBatch() evidence=%q err=%v", evidence, err)
	}
	committed, err = adapter.Reconcile(context.Background(), "contract-batch", payload)
	if err != nil || !committed {
		t.Fatalf("reconciliation after commit committed=%v err=%v", committed, err)
	}
	replayedEvidence, err := adapter.CommitBatch(context.Background(), "contract-batch", payload)
	if err != nil || replayedEvidence != evidence {
		t.Fatalf("duplicate CommitBatch() evidence=%q err=%v", replayedEvidence, err)
	}
	conflicting := json.RawMessage(`{"batchId":"contract-batch","metadataRevision":"other","sourceDigest":"source","records":[],"commitEvidence":"other"}`)
	if _, err := adapter.CommitBatch(context.Background(), "contract-batch", conflicting); err == nil {
		t.Fatal("destination accepted different content under a committed batch identity")
	}
	committed, err = adapter.Reconcile(context.Background(), "contract-batch", payload)
	if err != nil || !committed {
		t.Fatalf("conflicting duplicate changed existing evidence: committed=%v err=%v", committed, err)
	}
}
