package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/metadata"
)

func TestLegacySecretReferenceFailsBeforeSourceAccess(t *testing.T) {
	t.Setenv("MIMIR_SYNTHETIC_SECRET", "synthetic-credential-canary")
	_, source, stateDir, _ := setupLocalRun(t, "legacy-credential", `{"id":1}`)
	raw := []byte(strings.ReplaceAll(
		strings.Replace(draftMetadata("legacy-credential"), `"classification":"internal",`,
			`"classification":"internal","secretReferences":[{"id":"credential","reference":"env://MIMIR_SYNTHETIC_SECRET"}],`, 1),
		`"status":"draft"`, `"status":"active"`))
	parsed, err := metadata.ParseBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	raw = parsed.Canonical
	if _, err := (fileMetadataRepository{stateDir: stateDir}).Activate(context.Background(), raw, ""); err != nil {
		t.Fatal(err)
	}
	event := auditEvent{
		EventID:   digest([]byte("metadata.activate\x00legacy-credential\x00" + digest(raw))),
		Operation: "metadata.activated", Actor: "local-user", MetadataRevision: "legacy-credential",
		OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := ensureAudit(filepath.Join(stateDir, "audit", "events.jsonl"), event); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(source, "orders.jsonl")); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		call func() error
	}{
		{"run", func() error {
			_, err := Run(context.Background(), Options{SourceRoot: source, StateDir: stateDir, Actor: "local-user", AllowLocal: true})
			return err
		}},
		{"discovery", func() error {
			_, err := Discover(context.Background(), DiscoveryOptions{SourceRoot: source, StateDir: stateDir, Actor: "local-user", AllowLocal: true, Disposition: "hold"})
			return err
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.call()
			var failure domain.Failure
			if !errors.As(err, &failure) || failure.Category != domain.FailureUnsupportedCapability ||
				strings.Contains(err.Error(), "synthetic-credential-canary") {
				t.Fatalf("legacy credential reference failed unsafely: %v", err)
			}
		})
	}
	for _, name := range []string{"plans", "data", "metadata/proposals"} {
		if _, err := os.Stat(filepath.Join(stateDir, name)); !os.IsNotExist(err) {
			t.Fatalf("unsupported credential operation created %s: %v", name, err)
		}
	}
}
