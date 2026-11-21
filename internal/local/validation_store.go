package local

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/metadata"
	"github.com/kumerian/mimir/internal/ports"
)

type fileMetadataValidationStore struct {
	stateDir string
}

func (s fileMetadataValidationStore) Capabilities() ports.CapabilitySet {
	return ports.CapabilitySet{
		ContractVersion: ports.MetadataValidationStoreVersion,
		Operations:      []string{"append_immutable_report", "load_by_input_digest"},
		Unsupported:     []string{"report_mutation", "distributed_transactions"},
		Concurrency:     1,
		MaximumBytes:    1 << 20,
		Classifications: []string{"public", "internal"},
	}
}

func (s fileMetadataValidationStore) Append(ctx context.Context, report metadata.ValidationReport) error {
	if err := ctx.Err(); err != nil {
		return domain.ContextFailure("metadata.validation.append", err)
	}
	if err := validateReportIdentity(report); err != nil {
		return err
	}
	raw, err := json.Marshal(report)
	if err != nil {
		return domain.NewFailure(domain.FailureInvalidInput, "metadata.validation.append", "validation report cannot be encoded")
	}
	if err := ensurePrivateDirectory(s.stateDir); err != nil {
		return domain.NewFailure(domain.FailureUnavailable, "metadata.validation.append", "validation report directory is unavailable")
	}
	lock, err := acquireLock(s.stateDir)
	if err != nil {
		return err
	}
	defer releaseLock(lock)
	reportDir := filepath.Join(s.stateDir, "metadata", "validation-reports")
	if err := ensurePrivateDirectory(reportDir); err != nil {
		return domain.NewFailure(domain.FailureUnavailable, "metadata.validation.append", "validation report directory is unavailable")
	}
	if err := writeImmutable(filepath.Join(reportDir, report.InputDigest+".json"), raw); err != nil {
		return err
	}
	return nil
}

func (s fileMetadataValidationStore) Load(ctx context.Context, inputDigest string) (metadata.ValidationReport, error) {
	if err := ctx.Err(); err != nil {
		return metadata.ValidationReport{}, domain.ContextFailure("metadata.validation.load", err)
	}
	if !validDigest(inputDigest) {
		return metadata.ValidationReport{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.validation.load", "validation report identity is invalid")
	}
	path := filepath.Join(s.stateDir, "metadata", "validation-reports", inputDigest+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return metadata.ValidationReport{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.validation.load", "validation report is unavailable")
	}
	var report metadata.ValidationReport
	if err := json.Unmarshal(raw, &report); err != nil || validateReportIdentity(report) != nil ||
		report.InputDigest != inputDigest {
		return metadata.ValidationReport{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.validation.load", "stored validation report failed integrity checks")
	}
	canonical, err := json.Marshal(report)
	if err != nil || string(canonical) != string(raw) {
		return metadata.ValidationReport{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.validation.load", "stored validation report is not canonical")
	}
	if !isPrivateFile(path) {
		return metadata.ValidationReport{}, domain.NewFailure(domain.FailureAccessDenied, "metadata.validation.load", "validation report permissions must be private to the local user")
	}
	return report, nil
}

func validateReportIdentity(report metadata.ValidationReport) error {
	if !metadata.SafeValidationReport(report) || !validDigest(report.InputDigest) {
		return domain.NewFailure(domain.FailureInvalidInput, "metadata.validation", "validation report contract or eligibility is invalid")
	}
	return nil
}

func validDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func ValidateMetadata(ctx context.Context, raw []byte, stateDir string) (metadata.ValidationReport, error) {
	return ValidateMetadataWithStore(ctx, raw, stateDir, nil)
}

func ValidateMetadataWithStore(ctx context.Context, raw []byte, stateDir string, store ports.MetadataValidationStore) (metadata.ValidationReport, error) {
	if stateDir == "" {
		return metadata.ValidationReport{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.validation", "validation report state directory is required")
	}
	if err := ctx.Err(); err != nil {
		return metadata.ValidationReport{}, domain.ContextFailure("metadata.validation", err)
	}
	report := metadata.Validate(raw)
	if store == nil {
		store = fileMetadataValidationStore{stateDir: stateDir}
	}
	if err := store.Append(ctx, report); err != nil {
		return metadata.ValidationReport{}, err
	}
	return report, nil
}

func LoadMetadataValidationReport(ctx context.Context, stateDir, inputDigest string) (metadata.ValidationReport, error) {
	return LoadMetadataValidationReportWithStore(ctx, stateDir, inputDigest, nil)
}

func LoadMetadataValidationReportWithStore(ctx context.Context, stateDir, inputDigest string, store ports.MetadataValidationStore) (metadata.ValidationReport, error) {
	if store == nil {
		store = fileMetadataValidationStore{stateDir: stateDir}
	}
	return store.Load(ctx, inputDigest)
}

var _ ports.MetadataValidationStore = fileMetadataValidationStore{}
