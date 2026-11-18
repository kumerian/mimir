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

func TestLocalActivationRejectsUnconsumedSecretReference(t *testing.T) {
	_, _, stateDir, candidate := setupLocalRun(t, "secret-ref", `{"id":1}`)
	input := strings.Replace(draftMetadata("secret-ref"), `"classification":"internal",`, `"classification":"internal","secretReferences":[{"id":"credential","reference":"env://MIMIR_SYNTHETIC_SECRET"}],`, 1)
	if err := os.WriteFile(candidate, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Activate(context.Background(), candidate, stateDir, "local-user", true)
	var failure domain.Failure
	if !errors.As(err, &failure) || failure.Category != domain.FailureUnsupportedCapability {
		t.Fatalf("unconsumed credential reference = %v, want unsupported capability", err)
	}
	if _, err := os.Stat(filepath.Join(stateDir, "metadata", "active.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("unsupported credential reference activated: %v", err)
	}
}
