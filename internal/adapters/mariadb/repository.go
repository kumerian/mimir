package mariadb

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/metadata"
)

func (s *Store) AppendDraft(ctx context.Context, raw json.RawMessage, actor string) error {
	if err := s.requireDB("metadata.repository.append"); err != nil {
		return err
	}
	parsed, err := metadata.Parse(bytes.NewReader(raw))
	if err != nil || parsed.Document.Status != "draft" || actor == "" {
		return domain.NewFailure(domain.FailureInvalidInput, "metadata.repository.append", "valid draft metadata and author identity are required")
	}
	document := parsed.Canonical
	digest := metadata.Digest(document)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return databaseFailure("metadata.repository.append", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO metadata_revisions
		(revision_id, revision_status, metadata_digest, document, actor)
		VALUES (?, 'draft', ?, ?, ?)
		ON DUPLICATE KEY UPDATE revision_id=VALUES(revision_id)`,
		parsed.Document.Revision, digest, string(document), actor); err != nil {
		return databaseFailure("metadata.repository.append", err)
	}
	var priorDigest, priorDocument string
	if err := tx.QueryRowContext(ctx, `SELECT metadata_digest, document FROM metadata_revisions
		WHERE revision_id = ? AND revision_status = 'draft' FOR UPDATE`,
		parsed.Document.Revision).Scan(&priorDigest, &priorDocument); err != nil {
		return databaseFailure("metadata.repository.append", err)
	}
	if priorDigest != digest || priorDocument != string(document) {
		return domain.NewFailure(domain.FailureConflict, "metadata.repository.append", "revision identity is immutable and already has different content")
	}
	if err := tx.Commit(); err != nil {
		return domain.NewFailure(domain.FailureAmbiguousCommit, "metadata.repository.append", "draft persistence outcome is uncertain; retry the same immutable revision")
	}
	return nil
}

func (s *Store) LoadDraft(ctx context.Context, revision string) (json.RawMessage, error) {
	if err := s.requireDB("metadata.repository.draft"); err != nil {
		return nil, err
	}
	if revision == "" || len(revision) > 128 {
		return nil, domain.NewFailure(domain.FailureInvalidInput, "metadata.repository.draft", "revision identifier is invalid")
	}
	var raw, expectedDigest string
	err := s.db.QueryRowContext(ctx, `SELECT document, metadata_digest FROM metadata_revisions
		WHERE revision_id = ? AND revision_status = 'draft'`, revision).Scan(&raw, &expectedDigest)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.NewFailure(domain.FailureInvalidInput, "metadata.repository.draft", "draft metadata is unavailable")
	}
	if err != nil {
		return nil, databaseFailure("metadata.repository.draft", err)
	}
	if err := validateStoredDocument([]byte(raw), expectedDigest, "draft", revision); err != nil {
		return nil, err
	}
	return json.RawMessage(raw), nil
}

func (s *Store) List(ctx context.Context) ([]json.RawMessage, error) {
	if err := s.requireDB("metadata.repository.list"); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT revision_id, revision_status, metadata_digest, document
		FROM metadata_revisions ORDER BY revision_id, revision_status`)
	if err != nil {
		return nil, databaseFailure("metadata.repository.list", err)
	}
	defer rows.Close()
	revisions := make([]json.RawMessage, 0)
	for rows.Next() {
		var revision, status, expectedDigest, raw string
		if err := rows.Scan(&revision, &status, &expectedDigest, &raw); err != nil {
			return nil, databaseFailure("metadata.repository.list", err)
		}
		if err := validateStoredDocument([]byte(raw), expectedDigest, status, revision); err != nil {
			return nil, err
		}
		revisions = append(revisions, json.RawMessage(raw))
	}
	if err := rows.Err(); err != nil {
		return nil, databaseFailure("metadata.repository.list", err)
	}
	return revisions, nil
}

func (s *Store) LoadActive(ctx context.Context) (json.RawMessage, error) {
	if err := s.requireDB("metadata.repository.active"); err != nil {
		return nil, err
	}
	var revision, pointerDigest, document, documentDigest string
	err := s.db.QueryRowContext(ctx, `SELECT a.revision_id, a.metadata_digest, r.document, r.metadata_digest
		FROM metadata_active a
		JOIN metadata_revisions r ON r.revision_id = a.revision_id AND r.revision_status = 'active'
		WHERE a.singleton_id = 1 AND a.revision_id <> ''`).Scan(
		&revision, &pointerDigest, &document, &documentDigest,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.NewFailure(domain.FailureInvalidInput, "metadata.active", "no active metadata revision is available")
	}
	if err != nil {
		return nil, databaseFailure("metadata.repository.active", err)
	}
	if pointerDigest != documentDigest {
		return nil, domain.NewFailure(domain.FailureInvalidInput, "metadata.active", "active metadata pointer failed its digest check")
	}
	if err := validateStoredDocument([]byte(document), pointerDigest, "active", revision); err != nil {
		return nil, err
	}
	return json.RawMessage(document), nil
}

func (s *Store) Activate(ctx context.Context, raw json.RawMessage, expectedActive string) (string, error) {
	if err := s.requireDB("metadata.repository.activate"); err != nil {
		return "", err
	}
	parsed, err := metadata.Parse(bytes.NewReader(raw))
	if err != nil || parsed.Document.Status != "active" || !bytes.Equal(parsed.Canonical, raw) {
		return "", domain.NewFailure(domain.FailureInvalidInput, "metadata.repository.activate", "canonical active metadata is required")
	}
	digest := metadata.Digest(raw)
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return "", databaseFailure("metadata.repository.activate", err)
	}
	defer tx.Rollback()

	var previousRevision, previousDigest string
	err = tx.QueryRowContext(ctx, `SELECT revision_id, metadata_digest FROM metadata_active
		WHERE singleton_id = 1 FOR UPDATE`).Scan(&previousRevision, &previousDigest)
	if err != nil {
		return "", databaseFailure("metadata.repository.activate", err)
	}
	if previousRevision == parsed.Document.Revision && previousDigest == digest {
		var committedPrevious string
		err := tx.QueryRowContext(ctx, `SELECT previous_revision_id FROM metadata_activation_history
			WHERE revision_id = ? AND metadata_digest = ? ORDER BY id DESC LIMIT 1`,
			parsed.Document.Revision, digest).Scan(&committedPrevious)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return "", databaseFailure("metadata.repository.activate", err)
		}
		if err == nil && committedPrevious == expectedActive {
			return committedPrevious, nil
		}
	}
	if previousRevision != "" && previousRevision != parsed.Document.Revision && expectedActive == "" {
		return "", domain.NewFailure(domain.FailureInvalidInput, "metadata.repository.activate", "expected active revision is required when superseding an active revision")
	}
	if expectedActive != "" && expectedActive != previousRevision {
		return "", domain.NewFailure(domain.FailureConflict, "metadata.repository.activate", "active revision changed; candidate activation was not applied")
	}
	if previousRevision == parsed.Document.Revision && previousDigest == digest {
		return previousRevision, nil
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO metadata_revisions
		(revision_id, revision_status, metadata_digest, document, actor)
		VALUES (?, 'active', ?, ?, 'local-user')
		ON DUPLICATE KEY UPDATE revision_id=VALUES(revision_id)`,
		parsed.Document.Revision, digest, string(raw)); err != nil {
		return "", databaseFailure("metadata.repository.activate", err)
	}
	var storedDigest, storedDocument string
	if err := tx.QueryRowContext(ctx, `SELECT metadata_digest, document FROM metadata_revisions
		WHERE revision_id = ? AND revision_status = 'active' FOR UPDATE`,
		parsed.Document.Revision).Scan(&storedDigest, &storedDocument); err != nil {
		return "", databaseFailure("metadata.repository.activate", err)
	}
	if storedDigest != digest || storedDocument != string(raw) {
		return "", domain.NewFailure(domain.FailureConflict, "metadata.repository.activate", "revision identity is immutable and already has different content")
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO metadata_activation_history
		(revision_id, metadata_digest, previous_revision_id) VALUES (?, ?, ?)`,
		parsed.Document.Revision, digest, previousRevision); err != nil {
		return "", databaseFailure("metadata.repository.activate", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE metadata_active
		SET revision_id = ?, metadata_digest = ?, activated_at = ?
		WHERE singleton_id = 1`, parsed.Document.Revision, digest, time.Now().UTC()); err != nil {
		return "", databaseFailure("metadata.repository.activate", err)
	}
	if err := tx.Commit(); err != nil {
		return "", domain.NewFailure(domain.FailureAmbiguousCommit, "metadata.repository.activate", "activation outcome is uncertain; reconcile the active revision before retrying")
	}
	return previousRevision, nil
}

func (s *Store) requireDB(operation string) error {
	if s == nil || s.db == nil {
		return domain.NewFailure(domain.FailureUnavailable, operation, "metadata database is unavailable")
	}
	return nil
}

func validateStoredDocument(raw []byte, expectedDigest, expectedStatus, expectedRevision string) error {
	if metadata.Digest(raw) != expectedDigest {
		return domain.NewFailure(domain.FailureInvalidInput, "metadata.repository", "stored metadata failed its digest check")
	}
	parsed, err := metadata.Parse(bytes.NewReader(raw))
	if err != nil || parsed.Document.Status != expectedStatus || parsed.Document.Revision != expectedRevision ||
		!bytes.Equal(parsed.Canonical, raw) {
		return domain.NewFailure(domain.FailureInvalidInput, "metadata.repository", "stored metadata failed validation")
	}
	return nil
}
