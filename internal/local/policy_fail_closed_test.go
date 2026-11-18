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

func TestActivationRejectsMissingOrUnknownClassification(t *testing.T) {
	for _, classification := range []string{"", `"classification":"unknown",`} {
		t.Run(classification, func(t *testing.T) {
			_, _, stateDir, candidate := setupLocalRun(t, "invalid-policy", `{"id":1}`)
			input := strings.Replace(draftMetadata("invalid-policy"), `"classification":"internal",`, classification, 1)
			if err := os.WriteFile(candidate, []byte(input), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := Activate(context.Background(), candidate, stateDir, "local-user", true)
			var failure domain.Failure
			if !errors.As(err, &failure) || failure.Category != domain.FailureInvalidInput {
				t.Fatalf("invalid policy = %v, want invalid input", err)
			}
			if _, err := os.Stat(filepath.Join(stateDir, "metadata", "active.jsonl")); !os.IsNotExist(err) {
				t.Fatalf("invalid policy produced activation: %v", err)
			}
		})
	}
}
