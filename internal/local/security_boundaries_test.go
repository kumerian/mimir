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
)

func TestDeniedControlOperationsAuditWithoutPersistingMetadataOrPlans(t *testing.T) {
	tests := []struct {
		name string
		call func(string, string) error
	}{
		{"author", func(candidate, state string) error {
			_, err := Author(context.Background(), candidate, state, "synthetic-secret-actor", false)
			return err
		}},
		{"activate", func(candidate, state string) error {
			_, err := Activate(context.Background(), candidate, state, "synthetic-secret-actor", false)
			return err
		}},
		{"run", func(_, state string) error {
			_, err := Run(context.Background(), Options{SourceRoot: "/unreadable", StateDir: state, Actor: "synthetic-secret-actor"})
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stateDir := filepath.Join(t.TempDir(), "state")
			err := tt.call("/unreadable-candidate", stateDir)
			var failure domain.Failure
			if !errors.As(err, &failure) || failure.Category != domain.FailureAccessDenied {
				t.Fatalf("denied operation = %v", err)
			}
			for _, name := range []string{"metadata", "plans", "state", "data"} {
				if _, err := os.Stat(filepath.Join(stateDir, name)); !os.IsNotExist(err) {
					t.Fatalf("denied operation created %s: %v", name, err)
				}
			}
			raw, err := os.ReadFile(filepath.Join(stateDir, "audit", "events.jsonl"))
			if err != nil || strings.Contains(string(raw), "synthetic-secret-actor") || !strings.Contains(string(raw), `"failureCategory":"access_denied"`) {
				t.Fatalf("denial audit missing or not redacted: %q, %v", raw, err)
			}
		})
	}
}

func TestLegacyRestrictedActivationCannotBeRunOrDiscovered(t *testing.T) {
	_, source, stateDir, _ := setupLocalRun(t, "legacy-restricted", `{"id":1}`)
	raw := []byte(strings.ReplaceAll(
		strings.Replace(draftMetadata("legacy-restricted"), `"id":"orders","path":"orders.jsonl"`, `"id":"orders","path":"orders.jsonl","classification":"restricted"`, 1),
		`"status":"draft"`, `"status":"active"`))
	if _, err := (fileMetadataRepository{stateDir: stateDir}).Activate(context.Background(), raw, ""); err != nil {
		t.Fatal(err)
	}
	event := auditEvent{
		EventID:   digest([]byte("metadata.activate\x00legacy-restricted\x00" + digest(raw))),
		Operation: "metadata.activated", Actor: "local-user", MetadataRevision: "legacy-restricted",
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
			var failure domain.Failure
			if err := test.call(); !errors.As(err, &failure) || failure.Category != domain.FailureAccessDenied {
				t.Fatalf("legacy policy did not fail before source read: %v", err)
			}
		})
	}
	for _, name := range []string{"plans", "data", "metadata/proposals"} {
		if _, err := os.Stat(filepath.Join(stateDir, name)); !os.IsNotExist(err) {
			t.Fatalf("restricted operation created %s: %v", name, err)
		}
	}
}
