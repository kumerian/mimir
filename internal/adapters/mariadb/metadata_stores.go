package mariadb

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/metadata"
	"github.com/kumerian/mimir/internal/ports"
)

func (s *Store) Append(ctx context.Context, report metadata.ValidationReport) error {
	if err := s.requireDB("metadata.validation.append"); err != nil {
		return err
	}
	if err := validateReport(report); err != nil {
		return err
	}
	raw, err := json.Marshal(report)
	if err != nil {
		return domain.NewFailure(domain.FailureInvalidInput, "metadata.validation.append", "validation report cannot be encoded")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return databaseFailure("metadata.validation.append", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO metadata_validation_reports (input_digest, report_json)
		VALUES (?, ?) ON DUPLICATE KEY UPDATE input_digest=VALUES(input_digest)`,
		report.InputDigest, string(raw)); err != nil {
		return databaseFailure("metadata.validation.append", err)
	}
	var stored string
	if err := tx.QueryRowContext(ctx, `SELECT report_json FROM metadata_validation_reports
		WHERE input_digest = ? FOR UPDATE`, report.InputDigest).Scan(&stored); err != nil {
		return databaseFailure("metadata.validation.append", err)
	}
	if stored != string(raw) {
		return domain.NewFailure(domain.FailureConflict, "metadata.validation.append", "validation report identity is immutable and already has different content")
	}
	if err := tx.Commit(); err != nil {
		return domain.NewFailure(domain.FailureAmbiguousCommit, "metadata.validation.append", "validation report persistence outcome is uncertain")
	}
	return nil
}

func (s *Store) Load(ctx context.Context, inputDigest string) (metadata.ValidationReport, error) {
	if err := s.requireDB("metadata.validation.load"); err != nil {
		return metadata.ValidationReport{}, err
	}
	if !validDigest(inputDigest) {
		return metadata.ValidationReport{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.validation.load", "validation report identity is invalid")
	}
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT report_json FROM metadata_validation_reports WHERE input_digest = ?`, inputDigest).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return metadata.ValidationReport{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.validation.load", "validation report is unavailable")
	}
	if err != nil {
		return metadata.ValidationReport{}, databaseFailure("metadata.validation.load", err)
	}
	var report metadata.ValidationReport
	if err := json.Unmarshal([]byte(raw), &report); err != nil ||
		report.InputDigest != inputDigest || validateReport(report) != nil {
		return metadata.ValidationReport{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.validation.load", "stored validation report failed integrity checks")
	}
	canonical, err := json.Marshal(report)
	if err != nil || !bytes.Equal(canonical, []byte(raw)) {
		return metadata.ValidationReport{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.validation.load", "stored validation report is not canonical")
	}
	return report, nil
}

func (s *Store) AppendProposal(ctx context.Context, proposalID string, raw json.RawMessage) error {
	if err := s.requireDB("metadata.proposal.append"); err != nil {
		return err
	}
	if proposalID == "" || len(raw) == 0 || len(raw) > 1<<20 || metadata.Digest(raw) != proposalID || !json.Valid(raw) {
		return domain.NewFailure(domain.FailureInvalidInput, "metadata.proposal.append", "proposal identity or content is invalid")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return databaseFailure("metadata.proposal.append", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO metadata_discovery_proposals (proposal_id, proposal_json)
		VALUES (?, ?) ON DUPLICATE KEY UPDATE proposal_id=VALUES(proposal_id)`, proposalID, string(raw)); err != nil {
		return databaseFailure("metadata.proposal.append", err)
	}
	var stored string
	if err := tx.QueryRowContext(ctx, `SELECT proposal_json FROM metadata_discovery_proposals
		WHERE proposal_id = ? FOR UPDATE`, proposalID).Scan(&stored); err != nil {
		return databaseFailure("metadata.proposal.append", err)
	}
	if stored != string(raw) {
		return domain.NewFailure(domain.FailureConflict, "metadata.proposal.append", "proposal identity is immutable and already has different content")
	}
	if err := tx.Commit(); err != nil {
		return domain.NewFailure(domain.FailureAmbiguousCommit, "metadata.proposal.append", "discovery proposal persistence outcome is uncertain")
	}
	return nil
}

func (s *Store) ValidationCapabilities() ports.CapabilitySet {
	return ports.CapabilitySet{
		ContractVersion: ports.MetadataValidationStoreVersion,
		Operations:      []string{"append_immutable_report", "load_by_input_digest"},
		Unsupported:     []string{"report_mutation", "distributed_transactions"},
		Concurrency:     8,
		MaximumBytes:    1 << 20,
		Classifications: []string{"public", "internal"},
	}
}

func (s *Store) ProposalCapabilities() ports.CapabilitySet {
	return ports.CapabilitySet{
		ContractVersion: ports.MetadataProposalStoreVersion,
		Operations:      []string{"append_immutable_proposal"},
		Unsupported:     []string{"proposal_queries", "proposal_mutation"},
		Concurrency:     8,
		MaximumBytes:    1 << 20,
		Classifications: []string{"public", "internal"},
	}
}

func validateReport(report metadata.ValidationReport) error {
	if !metadata.SafeValidationReport(report) || !validDigest(report.InputDigest) {
		return domain.NewFailure(domain.FailureInvalidInput, "metadata.validation", "validation report contract or identity is invalid")
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

var (
	_ ports.MetadataRepository      = (*Store)(nil)
	_ ports.MetadataValidationStore = validationStore{store: nil}
	_ ports.MetadataProposalStore   = proposalStore{store: nil}
)

type validationStore struct{ store *Store }
type proposalStore struct{ store *Store }

func (s validationStore) Capabilities() ports.CapabilitySet { return s.store.ValidationCapabilities() }
func (s validationStore) Append(ctx context.Context, report metadata.ValidationReport) error {
	return s.store.Append(ctx, report)
}
func (s validationStore) Load(ctx context.Context, digest string) (metadata.ValidationReport, error) {
	return s.store.Load(ctx, digest)
}

func (s proposalStore) Capabilities() ports.CapabilitySet { return s.store.ProposalCapabilities() }
func (s proposalStore) Append(ctx context.Context, proposalID string, raw json.RawMessage) error {
	return s.store.AppendProposal(ctx, proposalID, raw)
}

func (s *Store) MetadataStores() (ports.MetadataRepository, ports.MetadataValidationStore, ports.MetadataProposalStore) {
	return s, validationStore{store: s}, proposalStore{store: s}
}
