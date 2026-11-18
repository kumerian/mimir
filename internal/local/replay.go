package local

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/metadata"
	"github.com/kumerian/mimir/internal/planning"
	replaycontract "github.com/kumerian/mimir/internal/replay"
)

type ReplayResult struct {
	ReplayPlanID  string `json:"replayPlanId"`
	OriginalRunID string `json:"originalRunId"`
	Result        Result `json:"result"`
}

type injectedReplayCrash struct {
	stage string
}

// Replay prepares a new replay identity only when the original metadata,
// source snapshot, adapter capabilities, and ordered rule versions still match.
// Execution reconciles the original idempotent batch rather than duplicating it.
func Replay(ctx context.Context, options Options, originalRunID, reason string) (ReplayResult, error) {
	return replayWithFaultHook(ctx, options, originalRunID, reason, nil)
}

func replayWithFaultHook(ctx context.Context, options Options, originalRunID, reason string, fault func(string) bool) (ReplayResult, error) {
	hit := func(stage string) {
		if fault != nil && fault(stage) {
			panic(injectedReplayCrash{stage: stage})
		}
	}
	if err := ctx.Err(); err != nil {
		return ReplayResult{}, domain.ContextFailure("replay.prepare", err)
	}
	if !options.AllowLocal || options.Actor != "local-user" {
		if err := recordDenial(options.StateDir, options.Actor, "replay.prepare", "access_denied"); err != nil {
			return ReplayResult{}, domain.NewFailure(domain.FailureUnavailable, "audit.denial", "replay was denied and denial audit could not be persisted")
		}
		return ReplayResult{}, domain.NewFailure(domain.FailureAccessDenied, "replay.prepare", "explicit local-user authorization is required")
	}
	if options.StateDir == "" || options.SourceRoot == "" || len(originalRunID) != 64 {
		return ReplayResult{}, domain.NewFailure(domain.FailureInvalidInput, "replay.prepare", "state, source, and valid original run identity are required")
	}
	if _, err := hex.DecodeString(originalRunID); err != nil {
		return ReplayResult{}, domain.NewFailure(domain.FailureInvalidInput, "replay.prepare", "original run identity is malformed")
	}
	if !isPrivateDirectory(options.StateDir) {
		return ReplayResult{}, domain.NewFailure(domain.FailureInvalidInput, "replay.prepare", "private local state directory is unavailable")
	}
	var original persistedPlan
	if err := readStrictJSON(filepath.Join(options.StateDir, "plans", originalRunID+".json"), &original); err != nil {
		return ReplayResult{}, domain.NewFailure(domain.FailureInvalidInput, "replay.prepare", "original pinned plan is unavailable or invalid")
	}
	if original.RunID != originalRunID || original.Plan.ContractVersion != planning.RunPlanContractVersion ||
		original.SourceDigest == "" || original.Plan.MetadataRevision == "" || original.Plan.MetadataDigest == "" {
		return ReplayResult{}, domain.NewFailure(domain.FailureInvalidInput, "replay.prepare", "original plan lacks valid pinned inputs")
	}
	parsed, _, err := loadActiveMetadata(options.StateDir)
	if err != nil {
		return ReplayResult{}, err
	}
	if !parsed.Document.RunEligible() || parsed.Document.Revision != original.Plan.MetadataRevision ||
		canonicalMetadataDigest(parsed.Document) != original.Plan.MetadataDigest {
		return ReplayResult{}, domain.NewFailure(domain.FailureUnsupportedCapability, "replay.prepare", "original metadata revision is not the active pinned revision")
	}
	if len(parsed.Document.Sources) != 1 || len(parsed.Document.Sources[0].Objects) != 1 {
		return ReplayResult{}, domain.NewFailure(domain.FailureUnsupportedCapability, "replay.prepare", "local replay requires exactly one source and one object")
	}
	source, object := parsed.Document.Sources[0], parsed.Document.Sources[0].Objects[0]
	sourceAdapter := fileSourceAdapter{root: options.SourceRoot}
	_, sourceDigest, err := sourceAdapter.ReadBatch(ctx, object.Path)
	if err != nil {
		return ReplayResult{}, err
	}
	destinationAdapter := fileDestinationAdapter{path: filepath.Join(options.StateDir, "data", "destination.jsonl")}
	current, err := planning.PlanRun(parsed.Document, planning.Request{
		SourceID: source.ID, ObjectID: object.ID, Authorized: true, MaximumClass: "internal",
		Source: sourceAdapter.Capabilities(), Destination: destinationAdapter.Capabilities(),
	})
	if err != nil {
		return ReplayResult{}, err
	}
	originalIdentity, err := original.Plan.WithSourceDigest(original.SourceDigest)
	if err != nil || originalIdentity != originalRunID {
		return ReplayResult{}, domain.NewFailure(domain.FailureInvalidInput, "replay.prepare", "original run identity does not match its immutable plan")
	}
	currentRules := append(append([]string(nil), current.Transformations...), current.QualityRules...)
	originalRules := append(append([]string(nil), original.Plan.Transformations...), original.Plan.QualityRules...)
	contractPlan, err := replaycontract.Prepare(
		replaycontract.Plan{
			RunID: original.RunID, MetadataRevision: original.Plan.MetadataRevision,
			MetadataDigest: original.Plan.MetadataDigest, SourceDigest: original.SourceDigest,
			AdapterVersions: append([]string(nil), original.Plan.CapabilityVersions...), RuleVersions: originalRules,
		},
		replaycontract.Plan{
			RunID: original.RunID, MetadataRevision: current.MetadataRevision,
			MetadataDigest: current.MetadataDigest, SourceDigest: sourceDigest,
			AdapterVersions: append([]string(nil), current.CapabilityVersions...), RuleVersions: currentRules,
		},
		reason,
	)
	if err != nil {
		return ReplayResult{}, err
	}
	replayDir := filepath.Join(options.StateDir, "plans", "replays")
	if err := ensurePrivateDirectory(replayDir); err != nil {
		return ReplayResult{}, domain.NewFailure(domain.FailureUnavailable, "replay.persist", "replay plan directory is unavailable")
	}
	hit("before_replay_plan_write")
	if err := writeImmutable(filepath.Join(replayDir, contractPlan.RunID+".json"), mustJSON(contractPlan)); err != nil {
		return ReplayResult{}, err
	}
	hit("after_replay_plan_write")
	auditPath := filepath.Join(options.StateDir, "audit", "events.jsonl")
	event := auditEvent{
		EventID:   metadata.Digest([]byte("replay.prepared\x00" + contractPlan.RunID)),
		Operation: "replay.prepared", Actor: options.Actor, MetadataRevision: parsed.Document.Revision,
		RunID: original.RunID, OriginalRunID: original.RunID, PlanID: contractPlan.RunID,
		RecordCount: 0, OccurredAt: currentTime(),
	}
	hit("before_replay_audit")
	if err := ensureAudit(auditPath, event); err != nil {
		return ReplayResult{}, domain.NewFailure(domain.FailureUnavailable, "replay.audit", "replay plan is stored but its audit event is pending")
	}
	hit("after_replay_audit")
	options.ExpectedMetadataRevision = original.Plan.MetadataRevision
	options.ExpectedMetadataDigest = original.Plan.MetadataDigest
	options.ExpectedSourceDigest = original.SourceDigest
	hit("before_replay_run_reconcile")
	result, err := Run(ctx, options)
	if err != nil {
		return ReplayResult{}, err
	}
	hit("after_replay_run_reconcile")
	return ReplayResult{ReplayPlanID: contractPlan.RunID, OriginalRunID: original.RunID, Result: result}, nil
}

func readStrictJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil || len(data) > maxSourceBytes*2 {
		return domain.NewFailure(domain.FailureUnavailable, "plan.read", "pinned plan is unavailable")
	}
	if err := metadata.ValidateJSONDocument(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	return nil
}

func currentTime() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
