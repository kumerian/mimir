package contracttest

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/kumerian/mimir/internal/ports"
)

type memoryDestination struct {
	batches map[string]json.RawMessage
}

func (d *memoryDestination) Capabilities() ports.DestinationCapabilities {
	return ports.DestinationCapabilities{Version: "memory/v1", IdempotentWrites: true, DurableCommitEvidence: true, Reconcile: true}
}

func (d *memoryDestination) CommitBatch(_ context.Context, id string, payload json.RawMessage) (string, error) {
	if d.batches == nil {
		d.batches = make(map[string]json.RawMessage)
	}
	if prior, exists := d.batches[id]; exists {
		if string(prior) != string(payload) {
			return "", fmt.Errorf("conflicting batch identity")
		}
		return "evidence", nil
	}
	d.batches[id] = append([]byte(nil), payload...)
	return "evidence", nil
}

func (d *memoryDestination) Reconcile(_ context.Context, id string, payload json.RawMessage) (bool, error) {
	prior, ok := d.batches[id]
	return ok && string(prior) == string(payload), nil
}

type secondMemoryDestination struct{ memoryDestination }

func (d *secondMemoryDestination) Capabilities() ports.DestinationCapabilities {
	return ports.DestinationCapabilities{Version: "memory-alt/v1", IdempotentWrites: true, DurableCommitEvidence: true, Reconcile: true}
}

func TestSharedDestinationContractRunsAgainstIndependentImplementations(t *testing.T) {
	t.Run("memory-a", func(t *testing.T) {
		RunDestinationContract(t, &memoryDestination{})
	})
	t.Run("memory-b", func(t *testing.T) {
		RunDestinationContract(t, &secondMemoryDestination{})
	})
}
