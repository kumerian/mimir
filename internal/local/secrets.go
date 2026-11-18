package local

import (
	"context"
	"errors"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/ports"
)

var localEnvReference = regexp.MustCompile(`^env://([A-Za-z_][A-Za-z0-9_]{0,127})$`)

const localSecretOperationTimeout = 5 * time.Second

type environmentSecretResolver struct{}

func (environmentSecretResolver) Capabilities() ports.CapabilitySet {
	return ports.CapabilitySet{
		ContractVersion: ports.SecretResolverVersion,
		Operations:      []string{"bounded_env_reference"},
		Unsupported:     []string{"cloud_secret_store", "ambient_credentials", "secret_export"},
		Concurrency:     1,
		Classifications: []string{"public", "internal"},
	}
}

// WithSecret scopes the environment value to one callback and clears the
// temporary byte buffer before returning. The local resolver is not production
// identity or secret management.
func (environmentSecretResolver) WithSecret(ctx context.Context, reference, operation string, use func(context.Context, []byte) error) error {
	if err := contextFailure(ctx.Err(), "secret.resolve"); err != nil {
		return err
	}
	if strings.TrimSpace(operation) == "" || use == nil {
		return domain.NewFailure(domain.FailureInvalidInput, "secret.resolve", "bounded operation and consumer are required")
	}
	matches := localEnvReference.FindStringSubmatch(reference)
	if len(matches) != 2 {
		return domain.NewFailure(domain.FailureAccessDenied, "secret.resolve", "local resolver accepts only an explicit env://NAME reference")
	}
	value, ok := os.LookupEnv(matches[1])
	if !ok {
		return domain.NewFailure(domain.FailureUnavailable, "secret.resolve", "referenced local secret is unavailable")
	}
	secret := []byte(value)
	defer clear(secret)
	operationCtx, cancel := context.WithTimeout(ctx, localSecretOperationTimeout)
	defer cancel()
	useErr := use(operationCtx, secret)
	if err := contextFailure(operationCtx.Err(), "secret.operation"); err != nil {
		return err
	}
	if useErr != nil {
		if errors.Is(useErr, context.DeadlineExceeded) {
			return domain.NewFailure(domain.FailureTimeout, "secret.operation", "bounded secret operation timed out")
		}
		if errors.Is(useErr, context.Canceled) {
			return domain.NewFailure(domain.FailureCancelled, "secret.operation", "bounded secret operation was cancelled")
		}
		return domain.NewFailure(domain.FailureUnavailable, "secret.operation", "bounded secret operation failed")
	}
	return nil
}

func contextFailure(err error, operation string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return domain.NewFailure(domain.FailureTimeout, operation, "bounded secret operation timed out")
	}
	return domain.NewFailure(domain.FailureCancelled, operation, "secret operation was cancelled")
}

var _ ports.SecretResolver = environmentSecretResolver{}
