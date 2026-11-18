package local

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/ports"
)

type unsupportedDiscoveryProvider struct {
	called bool
}

func (p *unsupportedDiscoveryProvider) Capabilities() ports.CapabilitySet {
	return ports.CapabilitySet{ContractVersion: ports.DiscoveryProviderVersion}
}

func (p *unsupportedDiscoveryProvider) Observe(context.Context, string) (json.RawMessage, error) {
	p.called = true
	return nil, nil
}

func TestJSONLDiscoveryProviderReturnsRedactedNormalizedSchema(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "orders.jsonl")
	if err := os.WriteFile(path, []byte("{\"customer\":\"synthetic-canary\",\"total\":4}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	provider := jsonlDiscoveryProvider{root: root}
	raw, err := provider.Observe(context.Background(), "orders.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "synthetic-canary") || strings.Contains(string(raw), "customer\":") {
		t.Fatalf("normalized observation leaked record values: %s", raw)
	}
	var observation observedSchema
	if err := json.Unmarshal(raw, &observation); err != nil {
		t.Fatal(err)
	}
	if observation.RecordCount != 1 || observation.SourceDigest == "" || len(observation.Columns) != 2 {
		t.Fatalf("unexpected normalized observation: %+v", observation)
	}
	if provider.Capabilities().ContractVersion != ports.DiscoveryProviderVersion {
		t.Fatal("provider did not declare its versioned port")
	}
}

func TestJSONLDiscoveryProviderCancellationAndUnsupportedValuesFailClosed(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "orders.jsonl"), []byte("{\"value\":null}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	provider := jsonlDiscoveryProvider{root: root}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := provider.Observe(ctx, "orders.jsonl"); err == nil {
		t.Fatal("provider ignored cancellation")
	}
	_, err := provider.Observe(context.Background(), "orders.jsonl")
	failure, ok := err.(domain.Failure)
	if !ok || failure.Category != domain.FailureUnsupportedCapability {
		t.Fatalf("null-only schema failure=%#v, want explicit unsupported capability", err)
	}
}

func TestDiscoveryRejectsMissingCapabilityBeforeProviderSideEffects(t *testing.T) {
	provider := &unsupportedDiscoveryProvider{}
	_, err := observeWithProvider(context.Background(), provider, "orders.jsonl")
	failure, ok := err.(domain.Failure)
	if !ok || failure.Category != domain.FailureUnsupportedCapability || provider.called {
		t.Fatalf("unsupported provider result=%#v called=%v", err, provider.called)
	}
}
