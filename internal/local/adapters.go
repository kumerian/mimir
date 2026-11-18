package local

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/metadata"
	"github.com/kumerian/mimir/internal/ports"
)

type fileSourceAdapter struct {
	root string
}

func (a fileSourceAdapter) Capabilities() ports.SourceCapabilities {
	return ports.SourceCapabilities{
		Version: "local.source/v1", BoundedSnapshot: true, StableIdentity: true,
		SchemaObservation: true, Acknowledgement: false, MaximumBatchRecords: maxSourceRecords,
	}
}

func (a fileSourceAdapter) ReadBatch(ctx context.Context, relativePath string) ([]json.RawMessage, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", domain.ContextFailure("source.read", err)
	}
	path, err := safeSourcePath(a.root, relativePath)
	if err != nil {
		return nil, "", err
	}
	records, snapshot, err := readSnapshot(ctx, path)
	if err != nil {
		return nil, "", err
	}
	return records, digest(snapshot), nil
}

func (a fileSourceAdapter) Acknowledge(context.Context, string) error {
	return domain.NewFailure(domain.FailureUnsupportedCapability, "source.acknowledge", "local immutable file source does not support acknowledgement")
}

type fileDestinationAdapter struct {
	path string
}

func (a fileDestinationAdapter) Capabilities() ports.DestinationCapabilities {
	return ports.DestinationCapabilities{
		Version: "local.destination/v1", IdempotentWrites: true, DurableCommitEvidence: true,
		Reconcile: true, MaximumBatchRecords: maxSourceRecords,
	}
}

func (a fileDestinationAdapter) Reconcile(ctx context.Context, batchID string, payload json.RawMessage) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, domain.ContextFailure("destination.reconcile", err)
	}
	var expected committedBatch
	if err := decodeStrict(payload, &expected); err != nil || expected.BatchID != batchID {
		return false, domain.NewFailure(domain.FailureInvalidInput, "destination.reconcile", "destination batch identity or payload is invalid")
	}
	committed, err := readAndRepairTail[committedBatch](a.path)
	if err != nil {
		return false, err
	}
	count := 0
	for _, prior := range committed {
		if prior.BatchID != batchID {
			continue
		}
		count++
		if prior.CommitEvidence != expected.CommitEvidence || !sameCommittedBatch(prior, expected) {
			return false, domain.NewFailure(domain.FailureConflict, "destination.reconcile", "batch identity conflicts with committed content")
		}
	}
	if count > 1 {
		return false, domain.NewFailure(domain.FailureConflict, "destination.reconcile", "multiple destination commits exist for the same batch identity")
	}
	return count == 1, nil
}

func (a fileDestinationAdapter) CommitBatch(ctx context.Context, batchID string, payload json.RawMessage) (string, error) {
	committed, err := a.Reconcile(ctx, batchID, payload)
	if err != nil {
		return "", err
	}
	var batch committedBatch
	if err := decodeStrict(payload, &batch); err != nil {
		return "", domain.NewFailure(domain.FailureInvalidInput, "destination.commit", "destination batch is invalid")
	}
	if committed {
		return batch.CommitEvidence, nil
	}
	if err := appendSynced(a.path, batch); err != nil {
		return "", domain.MapAdapterFailure(
			domain.FailureAmbiguousCommit,
			"destination.commit",
			"destination write outcome is uncertain; reconcile before retry",
			err,
		)
	}
	return batch.CommitEvidence, nil
}

func decodeStrict(raw []byte, target any) error {
	if err := metadata.ValidateJSONDocument(raw); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

var _ ports.SourceAdapter = fileSourceAdapter{}
var _ ports.DestinationAdapter = fileDestinationAdapter{}
