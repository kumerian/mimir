package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/kumerian/mimir/internal/domain"
)

func TestLocalProfileRejectsChangedOrMissingCapabilities(t *testing.T) {
	source := (fileSourceAdapter{}).Capabilities()
	destination := (fileDestinationAdapter{}).Capabilities()
	if err := validateLocalProfile(source, destination); err != nil {
		t.Fatalf("approved profile rejected: %v", err)
	}

	for _, test := range []struct {
		name string
		edit func()
	}{
		{"source version", func() { source.Version = "local.source/v2" }},
		{"destination version", func() { destination.Version = "local.destination/v2" }},
		{"reconciliation", func() { destination.Reconcile = false }},
		{"source limit", func() { source.MaximumBatchRecords = 0 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			source = (fileSourceAdapter{}).Capabilities()
			destination = (fileDestinationAdapter{}).Capabilities()
			test.edit()
			err := validateLocalProfile(source, destination)
			var failure domain.Failure
			if !errors.As(err, &failure) || failure.Category != domain.FailureUnsupportedCapability {
				t.Fatalf("profile mismatch = %v, want unsupported capability", err)
			}
		})
	}
}

func TestRunRejectsMissingSourceRootBeforeOpeningState(t *testing.T) {
	root := t.TempDir()
	_, err := Run(context.Background(), Options{
		SourceRoot: filepath.Join(root, "missing-source"),
		StateDir:   filepath.Join(root, "missing-state"),
		Actor:      "local-user",
		AllowLocal: true,
	})
	var failure domain.Failure
	if !errors.As(err, &failure) || failure.Category != domain.FailureInvalidInput || failure.Operation != "run.config" {
		t.Fatalf("invalid source configuration = %v, want run.config rejection", err)
	}
}

func TestRunRejectsMissingStateDirectoryAsInvalidConfiguration(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatal(err)
	}
	stateDir := filepath.Join(root, "missing-state")
	_, err := Run(context.Background(), Options{
		SourceRoot: source,
		StateDir:   stateDir,
		Actor:      "local-user",
		AllowLocal: true,
	})
	var failure domain.Failure
	if !errors.As(err, &failure) || failure.Category != domain.FailureInvalidInput || failure.Operation != "run.config" {
		t.Fatalf("invalid state configuration = %v, want run.config rejection", err)
	}
	if _, err := os.Stat(stateDir); !os.IsNotExist(err) {
		t.Fatalf("preflight created invalid state directory: %v", err)
	}
}
