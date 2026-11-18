package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kumerian/mimir/internal/domain"
)

func TestActivationRejectsRestrictedEffectiveObjectAndColumnBeforeActivation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"object", `"id":"orders","path":"orders.jsonl","classification":"restricted"`},
		{"column", `"id":"orders","path":"orders.jsonl","columns":[{"id":"id","type":"integer","classification":"confidential"}]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, stateDir, candidate := setupLocalRun(t, "denied-"+tt.name, `{"id":1}`)
			input := strings.Replace(draftMetadata("denied-"+tt.name), `"id":"orders","path":"orders.jsonl"`, tt.body, 1)
			if err := os.WriteFile(candidate, []byte(input), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := Activate(context.Background(), candidate, stateDir, "local-user", true)
			var failure domain.Failure
			if !errors.As(err, &failure) || failure.Category != domain.FailureAccessDenied {
				t.Fatalf("activation error = %v, want access denied", err)
			}
			if _, err := os.Stat(filepath.Join(stateDir, "metadata", "active.jsonl")); !os.IsNotExist(err) {
				t.Fatalf("denied activation changed active metadata: %v", err)
			}
			audits, err := readAndRepairTail[auditEvent](filepath.Join(stateDir, "audit", "events.jsonl"))
			if err != nil || len(audits) != 1 || audits[0].Operation != "access.denied" || audits[0].FailureCategory != "access_denied" {
				t.Fatalf("missing redacted denial audit: %+v, %v", audits, err)
			}
		})
	}
}
