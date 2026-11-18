package local

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kumerian/mimir/internal/domain"
)

func TestEnvironmentSecretResolverScopesAndClearsValue(t *testing.T) {
	t.Setenv("MIMIR_TEST_SECRET", "synthetic-canary")
	var retained []byte
	err := (environmentSecretResolver{}).WithSecret(context.Background(), "env://MIMIR_TEST_SECRET", "source.read", func(_ context.Context, value []byte) error {
		if string(value) != "synthetic-canary" {
			t.Fatal("resolver did not supply the referenced synthetic value")
		}
		retained = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range retained {
		if value != 0 {
			t.Fatal("resolver did not clear the temporary secret buffer")
		}
	}
}

func TestEnvironmentSecretResolverFailsClosedAndRedactsErrors(t *testing.T) {
	t.Setenv("MIMIR_TEST_SECRET", "synthetic-canary")
	resolver := environmentSecretResolver{}
	if err := resolver.WithSecret(context.Background(), "https://example.invalid/secret", "source.read", func(context.Context, []byte) error { return nil }); err == nil {
		t.Fatal("resolver accepted an unsupported reference type")
	}
	err := resolver.WithSecret(context.Background(), "env://MIMIR_TEST_SECRET", "source.read", func(context.Context, []byte) error {
		return errors.New("synthetic-canary")
	})
	if err == nil || strings.Contains(err.Error(), "synthetic-canary") {
		t.Fatalf("resolver leaked callback error: %v", err)
	}
	failure, ok := err.(domain.Failure)
	if !ok || failure.Category != domain.FailureUnavailable {
		t.Fatalf("callback failure=%#v, want stable unavailable category", err)
	}
	err = resolver.WithSecret(context.Background(), "env://MIMIR_MISSING_SECRET", "source.read", func(context.Context, []byte) error {
		t.Fatal("resolver called consumer for a missing reference")
		return nil
	})
	failure, ok = err.(domain.Failure)
	if !ok || failure.Category != domain.FailureUnavailable {
		t.Fatalf("missing reference failure=%#v, want stable unavailable category", err)
	}
}

func TestEnvironmentSecretResolverHonorsCancellationAfterUse(t *testing.T) {
	t.Setenv("MIMIR_TEST_SECRET", "synthetic-canary")
	ctx, cancel := context.WithCancel(context.Background())
	err := (environmentSecretResolver{}).WithSecret(ctx, "env://MIMIR_TEST_SECRET", "source.read", func(_ context.Context, _ []byte) error {
		cancel()
		return errors.New("operation stopped")
	})
	failure, ok := err.(domain.Failure)
	if !ok || failure.Category != domain.FailureCancelled {
		t.Fatalf("cancelled secret operation failure=%#v, want cancelled category", err)
	}
}

func TestEnvironmentSecretResolverProvidesBoundedContextAndMapsTimeout(t *testing.T) {
	t.Setenv("MIMIR_TEST_SECRET", "synthetic-canary")
	resolver := environmentSecretResolver{}
	err := resolver.WithSecret(context.Background(), "env://MIMIR_TEST_SECRET", "source.read", func(ctx context.Context, _ []byte) error {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > localSecretOperationTimeout {
			t.Fatalf("secret callback context has invalid deadline: %v, present=%v", deadline, ok)
		}
		return context.DeadlineExceeded
	})
	failure, ok := err.(domain.Failure)
	if !ok || failure.Category != domain.FailureTimeout || strings.Contains(err.Error(), "synthetic-canary") {
		t.Fatalf("timed-out secret operation=%#v, want redacted timeout", err)
	}

	expired, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	called := false
	err = resolver.WithSecret(expired, "env://MIMIR_TEST_SECRET", "source.read", func(context.Context, []byte) error {
		called = true
		return nil
	})
	failure, ok = err.(domain.Failure)
	if called || !ok || failure.Category != domain.FailureTimeout {
		t.Fatalf("expired caller context called=%v failure=%#v, want timeout before use", called, err)
	}
}
