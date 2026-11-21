package local

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/metadata"
	"github.com/kumerian/mimir/internal/planning"
	"github.com/kumerian/mimir/internal/ports"
	"github.com/kumerian/mimir/internal/quality"
	"github.com/kumerian/mimir/internal/state"
	"github.com/kumerian/mimir/internal/transform"
)

const (
	maxSourceBytes   = 16 << 20
	maxSourceRecords = 1000
)

type Options struct {
	SourceRoot               string
	AllowedSourceRoot        string
	StateDir                 string
	Actor                    string
	AllowLocal               bool
	ExpectedMetadataRevision string
	ExpectedMetadataDigest   string
	ExpectedSourceDigest     string
	MetadataRepository       ports.MetadataRepository
}

type Result struct {
	RunID            string `json:"runId"`
	MetadataRevision string `json:"metadataRevision"`
	RecordCount      int    `json:"recordCount"`
	AlreadyCommitted bool   `json:"alreadyCommitted"`
}

type committedBatch struct {
	BatchID          string                          `json:"batchId"`
	MetadataRevision string                          `json:"metadataRevision"`
	SourceDigest     string                          `json:"sourceDigest"`
	Records          []json.RawMessage               `json:"records"`
	OriginalRecords  []json.RawMessage               `json:"originalRecords,omitempty"`
	TransformProof   [][]transform.OperationEvidence `json:"transformProvenance,omitempty"`
	QualityResults   []quality.Result                `json:"qualityResults,omitempty"`
	Disposition      string                          `json:"disposition,omitempty"`
	CommitEvidence   string                          `json:"commitEvidence"`
}

type quarantinedRecord struct {
	OriginalRecord    json.RawMessage               `json:"originalRecord"`
	TransformedRecord json.RawMessage               `json:"transformedRecord"`
	TransformProof    []transform.OperationEvidence `json:"transformProvenance"`
	QualityResult     quality.Result                `json:"qualityResult"`
}

type quarantineBatch struct {
	BatchID        string              `json:"batchId"`
	SourceDigest   string              `json:"sourceDigest"`
	CommitEvidence string              `json:"commitEvidence"`
	Disposition    string              `json:"disposition"`
	Records        []quarantinedRecord `json:"records"`
}

type checkpoint struct {
	RunID          string `json:"runId"`
	SourceDigest   string `json:"sourceDigest"`
	CommitEvidence string `json:"commitEvidence"`
	CommittedAt    string `json:"committedAt"`
}

type auditEvent struct {
	EventID            string `json:"eventId"`
	Operation          string `json:"operation"`
	AttemptedOperation string `json:"attemptedOperation,omitempty"`
	FailureCategory    string `json:"failureCategory,omitempty"`
	FailureStage       string `json:"failureStage,omitempty"`
	CorrectionID       string `json:"correctionId,omitempty"`
	CorrectionVersion  string `json:"correctionVersion,omitempty"`
	InputDigest        string `json:"inputDigest,omitempty"`
	OutputDigest       string `json:"outputDigest,omitempty"`
	Actor              string `json:"actor"`
	MetadataRevision   string `json:"metadataRevision"`
	PreviousRevision   string `json:"previousRevision,omitempty"`
	RunID              string `json:"runId"`
	PlanID             string `json:"planId,omitempty"`
	Attempt            int    `json:"attempt,omitempty"`
	OriginalRunID      string `json:"originalRunId,omitempty"`
	RecordCount        int    `json:"recordCount"`
	OccurredAt         string `json:"occurredAt"`
}

type persistedPlan struct {
	Plan         planning.Plan `json:"plan"`
	RunID        string        `json:"runId"`
	SourceDigest string        `json:"sourceDigest"`
}

type injectedRunCrash struct {
	stage string
}

// Run executes one bounded immutable file snapshot. The development profile
// intentionally supports one process per StateDir and only an explicit
// local-user grant; production authorization is not inferred.
func Run(ctx context.Context, options Options) (result Result, runErr error) {
	return runWithFaultHook(ctx, options, nil)
}

// runWithFaultHook models process loss at named durable boundaries. A hook
// returning true closes, but does not remove, the ownership file and prevents
// normal failure transitions, matching a process that disappeared abruptly.
func runWithFaultHook(ctx context.Context, options Options, fault func(string) bool) (result Result, runErr error) {
	return runWithHooks(ctx, options, fault, nil)
}

func runWithAuditFailure(ctx context.Context, options Options, fail func(string) bool) (result Result, runErr error) {
	return runWithHooks(ctx, options, nil, fail)
}

func runWithHooks(ctx context.Context, options Options, fault, auditFailure func(string) bool) (result Result, runErr error) {
	var statePath string
	var stateStore fileRunStateStore
	var currentState state.Event
	crashInjected := false
	hit := func(stage string) {
		if fault != nil && fault(stage) {
			crashInjected = true
			panic(injectedRunCrash{stage: stage})
		}
	}
	defer func() {
		if crashInjected {
			return
		}
		if currentState.Status != state.Running || runErr == nil {
			return
		}
		failure, ok := runErr.(domain.Failure)
		if !ok {
			failure = domain.NewFailure(domain.FailureInternal, "run.execute", "run failed with an unclassified error")
		}
		next := state.Failed
		if failure.Category == domain.FailureCancelled {
			next = state.Cancelled
		}
		failureStage := failure.Stage
		if failureStage == "" {
			failureStage = failure.Operation
		}
		transition, err := state.Transition(currentState, currentState.OwnerToken, next, failure.Category, time.Now().UTC(), failureStage)
		if err == nil {
			err = stateStore.AppendTransition(context.Background(), currentState.RunID, mustJSON(transition))
		}
		if err != nil {
			runErr = domain.NewFailure(domain.FailureUnavailable, "run.state", "run outcome is known but its terminal state could not be durably recorded")
		}
	}()
	if err := ctx.Err(); err != nil {
		return Result{}, domain.ContextFailure("run.start", err)
	}
	if options.SourceRoot == "" || options.StateDir == "" {
		return Result{}, domain.NewFailure(domain.FailureInvalidInput, "run.plan", "source root and state directory are required")
	}
	if !options.AllowLocal || options.Actor != "local-user" {
		if err := recordDenial(options.StateDir, options.Actor, "run.authorize", "access_denied"); err != nil {
			return Result{}, domain.NewFailure(domain.FailureUnavailable, "audit.denial", "access was denied and denial audit could not be persisted")
		}
		return Result{}, domain.NewFailure(domain.FailureAccessDenied, "run.authorize", "explicit local-user authorization is required")
	}
	sourceAdapter := fileSourceAdapter{root: options.SourceRoot, allowedRoot: options.AllowedSourceRoot}
	destinationAdapter := fileDestinationAdapter{path: filepath.Join(options.StateDir, "data", "destination.jsonl")}
	sourceCapabilities := sourceAdapter.Capabilities()
	destinationCapabilities := destinationAdapter.Capabilities()
	if err := validateLocalProfile(sourceCapabilities, destinationCapabilities); err != nil {
		return Result{}, err
	}
	sourceInfo, err := os.Stat(options.SourceRoot)
	if err != nil || !sourceInfo.IsDir() {
		return Result{}, domain.NewFailure(domain.FailureInvalidInput, "run.config", "source root must be an accessible directory")
	}
	if !isPrivateDirectory(options.StateDir) {
		return Result{}, domain.NewFailure(domain.FailureInvalidInput, "run.config", "state directory must exist and be private to the local user")
	}
	lock, err := acquireLock(options.StateDir)
	if err != nil {
		return Result{}, err
	}
	defer func() {
		if crashInjected {
			_ = lock.file.Close()
			return
		}
		releaseLock(lock)
	}()

	parsed, _, err := loadActiveMetadataWithRepository(ctx, options.StateDir, options.MetadataRepository)
	if err != nil {
		return Result{}, err
	}
	if (options.ExpectedMetadataRevision != "" && parsed.Document.Revision != options.ExpectedMetadataRevision) ||
		(options.ExpectedMetadataDigest != "" && canonicalMetadataDigest(parsed.Document) != options.ExpectedMetadataDigest) {
		return Result{}, domain.NewFailure(domain.FailureConflict, "run.plan", "active metadata changed after replay preparation")
	}
	if !parsed.Document.RunEligible() {
		return Result{}, domain.NewFailure(domain.FailureInvalidInput, "run.plan", "only an activated metadata revision is eligible for a new run")
	}
	if err := localMetadataPolicy(parsed.Document); err != nil {
		if err := recordDenial(options.StateDir, options.Actor, "run.plan", "access_denied"); err != nil {
			return Result{}, domain.NewFailure(domain.FailureUnavailable, "audit.denial", "access was denied and denial audit could not be persisted")
		}
		return Result{}, domain.NewFailure(domain.FailureAccessDenied, "run.plan", "local reference profile permits only public or internal metadata")
	}
	if err := validateLocalMetadataCapabilities(parsed.Document); err != nil {
		return Result{}, err
	}
	if len(parsed.Document.Sources) != 1 || len(parsed.Document.Sources[0].Objects) != 1 {
		return Result{}, domain.NewFailure(domain.FailureUnsupportedCapability, "run.plan", "local reference source requires exactly one source and one object")
	}
	source := parsed.Document.Sources[0]
	object := source.Objects[0]
	plan, err := planning.PlanRun(parsed.Document, planning.Request{
		SourceID: source.ID, ObjectID: object.ID, Authorized: true, MaximumClass: "internal",
		Source: sourceCapabilities, Destination: destinationCapabilities,
	})
	if err != nil {
		if failure, ok := err.(domain.Failure); ok && failure.Category == domain.FailureAccessDenied {
			if auditErr := recordDenial(options.StateDir, options.Actor, "run.plan", "access_denied"); auditErr != nil {
				return Result{}, domain.NewFailure(domain.FailureUnavailable, "audit.denial", "access was denied and denial audit could not be persisted")
			}
		}
		return Result{}, err
	}

	records, sourceDigest, err := sourceAdapter.ReadBatch(ctx, object.Path)
	if err != nil {
		return Result{}, err
	}
	if options.ExpectedSourceDigest != "" && sourceDigest != options.ExpectedSourceDigest {
		return Result{}, domain.NewFailure(domain.FailureConflict, "source.snapshot", "source snapshot changed after replay preparation")
	}
	runID, err := plan.WithSourceDigest(sourceDigest)
	if err != nil {
		return Result{}, domain.NewFailure(domain.FailureInternal, "run.plan", "stable run identity cannot be created")
	}
	planPath := filepath.Join(options.StateDir, "plans", runID+".json")
	if err := ensurePrivateDirectory(filepath.Dir(planPath)); err != nil {
		return Result{}, domain.NewFailure(domain.FailureUnavailable, "run.plan", "plan directory is unavailable")
	}
	if err := writeImmutable(planPath, mustJSON(persistedPlan{Plan: plan, RunID: runID, SourceDigest: sourceDigest})); err != nil {
		return Result{}, err
	}
	auditPath := filepath.Join(options.StateDir, "audit", "events.jsonl")
	telemetry := fileTelemetry{path: filepath.Join(options.StateDir, "telemetry", "events.jsonl")}
	emitTelemetry := func(name, stage, outcome string, attempt int, category string) {
		event := telemetryEvent{
			EventID: digest([]byte(strings.Join([]string{name, runID, fmt.Sprint(attempt)}, "\x00"))),
			Name:    name, Stage: stage, Outcome: outcome, FailureCategory: category,
			MetadataRevision: parsed.Document.Revision, PlanID: plan.PlanID, RunID: runID,
			Attempt: attempt, OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
		}
		if raw, marshalErr := json.Marshal(event); marshalErr == nil {
			_ = telemetry.Emit(context.Background(), raw)
		}
	}
	plannedEvent := auditEvent{
		EventID: digest([]byte("run.planned\x00" + runID)), Operation: "run.planned",
		Actor: options.Actor, MetadataRevision: parsed.Document.Revision, RunID: runID,
		PlanID: plan.PlanID, RecordCount: len(records), OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	hit("before_plan_audit")
	if err := ensureAudit(auditPath, plannedEvent); err != nil {
		return Result{}, domain.NewFailure(domain.FailureUnavailable, "audit.append", "run plan is stored but its audit intent is pending")
	}
	emitTelemetry("run.planned", "planning", "planned", 0, "")
	hit("after_plan_audit")
	statePath = filepath.Join(options.StateDir, "state", "runs.jsonl")
	if err := ensurePrivateDirectory(filepath.Dir(statePath)); err != nil {
		return Result{}, domain.NewFailure(domain.FailureUnavailable, "run.state", "run-state directory is unavailable")
	}
	stateStore = fileRunStateStore{path: statePath}
	rawHistory, err := stateStore.LoadHistory(ctx, runID)
	if err != nil {
		return Result{}, err
	}
	stateEvents := make([]state.Event, 0, len(rawHistory))
	for _, raw := range rawHistory {
		var event state.Event
		if err := decodeStrict(raw, &event); err != nil {
			return Result{}, domain.NewFailure(domain.FailureInvalidInput, "run.state", "persisted run history is invalid")
		}
		stateEvents = append(stateEvents, event)
	}
	ownerToken := digest([]byte(fmt.Sprintf("%s\x00%d\x00%d", runID, os.Getpid(), time.Now().UTC().UnixNano())))
	currentState, started, err := state.StartAttempt(stateEvents, runID, plan.PlanID, ownerToken, time.Now().UTC())
	if err != nil {
		return Result{}, err
	}
	if started {
		hit("before_state_start")
		if err := stateStore.AppendTransition(ctx, runID, mustJSON(currentState)); err != nil {
			return Result{}, err
		}
		emitTelemetry("run.started", "execution", "started", currentState.Attempt, "")
		hit("after_state_started")
	}
	outputRecords, transformProof, qualityResults, disposition, err := processRecords(ctx, records, object)
	if err != nil {
		if failure, ok := err.(domain.Failure); ok {
			if auditErr := recordRunFailure(auditPath, options.Actor, parsed.Document.Revision, plan.PlanID, runID, currentState.Attempt, len(records), failure); auditErr != nil {
				return Result{}, domain.NewFailure(domain.FailureUnavailable, "audit.append", "run failure occurred but its required audit event is pending")
			}
			emitTelemetry("run.failed", failure.Operation, "failed", currentState.Attempt, string(failure.Category))
		}
		return Result{}, err
	}
	publishedRecords := outputRecords
	publishedOriginals := records
	publishedProof := transformProof
	publishedQuality := qualityResults
	quarantined := make([]quarantinedRecord, 0)
	if disposition == "quarantine" {
		publishedRecords = make([]json.RawMessage, 0, len(records))
		publishedOriginals = make([]json.RawMessage, 0, len(records))
		publishedProof = make([][]transform.OperationEvidence, 0, len(records))
		publishedQuality = make([]quality.Result, 0, len(records))
		for i, result := range qualityResults {
			if result.Passed {
				publishedRecords = append(publishedRecords, outputRecords[i])
				publishedOriginals = append(publishedOriginals, records[i])
				publishedProof = append(publishedProof, transformProof[i])
				publishedQuality = append(publishedQuality, result)
				continue
			}
			quarantined = append(quarantined, quarantinedRecord{
				OriginalRecord: records[i], TransformedRecord: outputRecords[i],
				TransformProof: transformProof[i], QualityResult: result,
			})
		}
	}
	batch := committedBatch{
		BatchID:          runID,
		MetadataRevision: parsed.Document.Revision,
		SourceDigest:     sourceDigest,
		Records:          publishedRecords,
		OriginalRecords:  publishedOriginals,
		TransformProof:   publishedProof,
		QualityResults:   publishedQuality,
		Disposition:      disposition,
	}
	batchBody, err := json.Marshal(struct {
		Batch      committedBatch      `json:"batch"`
		Quarantine []quarantinedRecord `json:"quarantine"`
	}{batch, quarantined})
	if err != nil {
		return Result{}, domain.NewFailure(domain.FailureInternal, "run.plan", "processing evidence cannot be encoded")
	}
	evidenceBytes, err := json.Marshal(struct {
		BatchID      string `json:"batchId"`
		SourceDigest string `json:"sourceDigest"`
		Count        int    `json:"count"`
		ResultDigest string `json:"resultDigest"`
	}{runID, sourceDigest, len(records), metadata.Digest(batchBody)})
	if err != nil {
		return Result{}, domain.NewFailure(domain.FailureInternal, "run.plan", "unable to create commit identity")
	}
	batch.CommitEvidence = digest(evidenceBytes)
	quarantineEntry := quarantineBatch{
		BatchID: runID, SourceDigest: sourceDigest, CommitEvidence: batch.CommitEvidence,
		Disposition: disposition, Records: quarantined,
	}

	dataPath := destinationAdapter.path
	if err := ensurePrivateDirectory(filepath.Dir(dataPath)); err != nil {
		return Result{}, domain.NewFailure(domain.FailureUnavailable, "destination.open", "destination directory is unavailable")
	}
	batchPayload, err := json.Marshal(batch)
	if err != nil {
		return Result{}, domain.NewFailure(domain.FailureInternal, "destination.commit", "destination batch cannot be encoded")
	}
	destinationCommitted, err := destinationAdapter.Reconcile(ctx, batch.BatchID, batchPayload)
	if err != nil {
		return Result{}, err
	}
	alreadyCommitted := destinationCommitted
	quarantinePath := filepath.Join(options.StateDir, "data", "quarantine.jsonl")
	quarantineEntries, err := readAndRepairTail[quarantineBatch](quarantinePath)
	if err != nil {
		return Result{}, err
	}
	quarantineExists := false
	quarantineCount := 0
	for _, prior := range quarantineEntries {
		if prior.BatchID != runID {
			continue
		}
		quarantineCount++
		if prior.CommitEvidence != batch.CommitEvidence || !sameQuarantineBatch(prior, quarantineEntry) {
			return Result{}, domain.NewFailure(domain.FailureConflict, "destination.quarantine", "quarantine identity conflicts with committed content")
		}
		quarantineExists = true
	}
	if quarantineCount > 1 {
		return Result{}, domain.NewFailure(domain.FailureConflict, "destination.quarantine", "multiple quarantine commits exist for the same batch identity")
	}
	if len(quarantined) == 0 && quarantineExists {
		return Result{}, domain.NewFailure(domain.FailureConflict, "destination.quarantine", "unexpected quarantine evidence exists for this batch")
	}
	if quarantineExists && !destinationCommitted {
		return Result{}, domain.NewFailure(domain.FailureConflict, "destination.reconcile", "quarantine commit exists without its destination commit; preserve state for operator recovery")
	}
	alreadyCommitted = destinationCommitted && (len(quarantined) == 0 || quarantineExists)
	checkpointPath := filepath.Join(options.StateDir, "state", "checkpoints.jsonl")
	if err := ensurePrivateDirectory(filepath.Dir(checkpointPath)); err != nil {
		return Result{}, domain.NewFailure(domain.FailureUnavailable, "state.checkpoint", "checkpoint directory is unavailable")
	}
	checkpoints, err := readAndRepairTail[checkpoint](checkpointPath)
	if err != nil {
		return Result{}, err
	}
	checkpointExists := false
	checkpointCount := 0
	for _, prior := range checkpoints {
		if prior.RunID == runID {
			checkpointCount++
			if prior.CommitEvidence != batch.CommitEvidence || prior.SourceDigest != sourceDigest {
				return Result{}, domain.NewFailure(domain.FailureConflict, "state.checkpoint", "checkpoint conflicts with destination evidence")
			}
			checkpointExists = true
		}
	}
	if checkpointCount > 1 {
		return Result{}, domain.NewFailure(domain.FailureConflict, "state.checkpoint", "multiple checkpoints exist for the same run identity")
	}
	if checkpointExists && !alreadyCommitted {
		return Result{}, domain.NewFailure(domain.FailureConflict, "state.reconcile", "checkpoint claims a destination commit that is not present; preserve state for operator recovery")
	}
	if !destinationCommitted {
		hit("before_destination_commit")
		evidence, err := destinationAdapter.CommitBatch(ctx, batch.BatchID, batchPayload)
		if err != nil {
			return Result{}, err
		}
		if evidence != batch.CommitEvidence {
			return Result{}, domain.NewFailure(domain.FailureConflict, "destination.commit", "destination returned mismatched commit evidence")
		}
		emitTelemetry("destination.committed", "publication", "committed", currentState.Attempt, "")
		hit("after_destination_commit")
	} else {
		emitTelemetry("destination.reconciled", "publication", "committed", currentState.Attempt, "")
	}
	if len(quarantined) > 0 && !quarantineExists {
		hit("before_quarantine_commit")
		if err := appendSynced(quarantinePath, quarantineEntry); err != nil {
			return Result{}, domain.NewFailure(domain.FailureAmbiguousCommit, "destination.quarantine", "quarantine write outcome is uncertain; rerun to reconcile")
		}
		hit("after_quarantine_commit")
	}

	// The destination's durable batch identity is the evidence required before
	// state can advance. The file source has no acknowledgement capability.
	if !checkpointExists {
		hit("before_checkpoint_commit")
		domainCheckpoint, err := domain.NewCheckpoint(
			runID, plan.PlanID, batch.BatchID, currentState.Attempt, batch.CommitEvidence,
		)
		if err != nil {
			return Result{}, err
		}
		entry := checkpoint{RunID: runID, SourceDigest: sourceDigest, CommitEvidence: batch.CommitEvidence, CommittedAt: time.Now().UTC().Format(time.RFC3339Nano)}
		if domainCheckpoint.Run.String() != entry.RunID || domainCheckpoint.Batch.String() != batch.BatchID {
			return Result{}, domain.NewFailure(domain.FailureInternal, "state.checkpoint", "checkpoint identity does not match the committed batch")
		}
		if err := appendSynced(checkpointPath, entry); err != nil {
			return Result{}, domain.NewFailure(domain.FailureUnavailable, "state.checkpoint", "destination is committed but checkpoint persistence failed; rerun to reconcile")
		}
		emitTelemetry("checkpoint.committed", "checkpoint", "committed", currentState.Attempt, "")
		hit("after_checkpoint_commit")
	}

	event := auditEvent{
		EventID: digest([]byte(strings.Join([]string{"run.committed", runID, fmt.Sprint(currentState.Attempt)}, "\x00"))), Operation: "run.committed",
		Actor: options.Actor, MetadataRevision: parsed.Document.Revision, RunID: runID,
		PlanID: plan.PlanID, Attempt: currentState.Attempt, RecordCount: len(records), OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	hit("before_run_audit")
	if auditFailure != nil && auditFailure("run.committed") {
		return Result{}, domain.NewFailure(domain.FailureUnavailable, "audit.append", "required run audit evidence is pending")
	}
	if err := ensureAudit(auditPath, event); err != nil {
		return Result{}, domain.NewFailure(domain.FailureUnavailable, "audit.append", "data and checkpoint are committed but required audit evidence is pending; rerun to reconcile")
	}
	emitTelemetry("run.committed", "audit", "committed", currentState.Attempt, "")
	hit("after_run_audit")
	if currentState.Status == state.Running {
		hit("before_terminal_state")
		completed, err := state.Transition(currentState, currentState.OwnerToken, state.Succeeded, "", time.Now().UTC())
		if err != nil {
			return Result{}, err
		}
		if err := stateStore.AppendTransition(ctx, runID, mustJSON(completed)); err != nil {
			return Result{}, domain.NewFailure(domain.FailureUnavailable, "run.state", "run is committed but terminal state persistence is pending; rerun to reconcile")
		}
		currentState = completed
		emitTelemetry("run.succeeded", "completion", "succeeded", currentState.Attempt, "")
		hit("after_terminal_state")
	}
	return Result{RunID: runID, MetadataRevision: parsed.Document.Revision, RecordCount: len(records), AlreadyCommitted: alreadyCommitted}, nil
}

func safeSourcePath(root, relative, allowedRoot string) (string, error) {
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return "", domain.MapAdapterFailure(domain.FailureInvalidInput, "source.open", "source root is invalid", err)
	}
	rootResolved, err := filepath.EvalSymlinks(rootPath)
	if err != nil {
		return "", domain.MapAdapterFailure(domain.FailureUnavailable, "source.open", "source root is unavailable", err)
	}
	if allowedRoot != "" {
		allowedRootPath, err := filepath.Abs(allowedRoot)
		if err != nil {
			return "", domain.MapAdapterFailure(domain.FailureInvalidInput, "source.open", "allowed source root is invalid", err)
		}
		allowedRootResolved, err := filepath.EvalSymlinks(allowedRootPath)
		if err != nil {
			return "", domain.MapAdapterFailure(domain.FailureUnavailable, "source.open", "allowed source root is unavailable", err)
		}
		rel, err := filepath.Rel(allowedRootResolved, rootResolved)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
			return "", domain.NewFailure(domain.FailureAccessDenied, "source.open", "source root resolves outside the allowed source directory")
		}
	}
	candidate := filepath.Join(rootResolved, filepath.FromSlash(relative))
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", domain.MapAdapterFailure(domain.FailureUnavailable, "source.open", "source object is unavailable", err)
	}
	rel, err := filepath.Rel(rootResolved, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", domain.NewFailure(domain.FailureAccessDenied, "source.open", "source object resolves outside the configured source root")
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", domain.MapAdapterFailure(domain.FailureUnavailable, "source.open", "source object is unavailable", err)
	}
	if !info.Mode().IsRegular() {
		return "", domain.NewFailure(domain.FailureInvalidInput, "source.open", "source object must be a regular file")
	}
	return resolved, nil
}

func readSnapshot(ctx context.Context, path string) ([]json.RawMessage, []byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, domain.MapAdapterFailure(domain.FailureUnavailable, "source.read", "source is unavailable", err)
	}
	defer file.Close()
	limited := &io.LimitedReader{R: file, N: maxSourceBytes + 1}
	var raw bytes.Buffer
	scanner := bufio.NewScanner(limited)
	scanner.Buffer(make([]byte, 4096), 256<<10)
	records := make([]json.RawMessage, 0)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return nil, nil, domain.ContextFailure("source.read", err)
		}
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		canonical, err := metadata.CanonicalizeRecord(line)
		if err != nil {
			return nil, nil, domain.NewFailure(domain.FailureInvalidInput, "source.read", "source contains an invalid JSON record")
		}
		records = append(records, canonical)
		if len(records) > maxSourceRecords {
			return nil, nil, domain.NewFailure(domain.FailureInvalidInput, "source.read", "source exceeds the 1000-record batch limit")
		}
		raw.Write(scanner.Bytes())
		raw.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, domain.NewFailure(domain.FailureInvalidInput, "source.read", "source contains an oversized record or cannot be read")
	}
	if limited.N == 0 {
		return nil, nil, domain.NewFailure(domain.FailureInvalidInput, "source.read", "source exceeds the 16 MiB snapshot limit")
	}
	if raw.Len() > maxSourceBytes {
		return nil, nil, domain.NewFailure(domain.FailureInvalidInput, "source.read", "source exceeds the 16 MiB snapshot limit")
	}
	if len(records) == 0 {
		return nil, nil, domain.NewFailure(domain.FailureInvalidInput, "source.read", "source snapshot contains no records")
	}
	return records, raw.Bytes(), nil
}

func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func canonicalMetadataDigest(document metadata.Document) string {
	canonical, err := metadata.CanonicalJSON(document)
	if err != nil {
		return ""
	}
	return metadata.Digest(canonical)
}

func sameCommittedBatch(left, right committedBatch) bool {
	left.CommitEvidence = ""
	right.CommitEvidence = ""
	leftBytes, leftErr := json.Marshal(left)
	rightBytes, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftBytes, rightBytes)
}

func sameQuarantineBatch(left, right quarantineBatch) bool {
	left.CommitEvidence = ""
	right.CommitEvidence = ""
	leftBytes, leftErr := json.Marshal(left)
	rightBytes, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftBytes, rightBytes)
}

func processRecords(ctx context.Context, records []json.RawMessage, object metadata.Object) ([]json.RawMessage, [][]transform.OperationEvidence, []quality.Result, string, error) {
	outputs := make([]json.RawMessage, 0, len(records))
	provenance := make([][]transform.OperationEvidence, 0, len(records))
	results := make([]quality.Result, 0, len(records))
	transformRules := make([]transform.Rule, 0, len(object.Transformations))
	for _, rule := range object.Transformations {
		var value any
		if len(rule.Value) > 0 {
			decoder := json.NewDecoder(bytes.NewReader(rule.Value))
			decoder.UseNumber()
			if err := decoder.Decode(&value); err != nil {
				return nil, nil, nil, "", domain.NewFailure(domain.FailureInvalidInput, "transform.rule", "transformation value is invalid")
			}
		}
		transformRules = append(transformRules, transform.Rule{
			ID: rule.ID, Version: rule.Version, Operation: rule.Operation, Field: rule.Field,
			TargetField: rule.TargetField, Value: value,
		})
	}
	qualityRules := make([]quality.Rule, 0, len(object.QualityRules))
	for _, rule := range object.QualityRules {
		qualityRules = append(qualityRules, quality.Rule{
			ID: rule.ID, Version: rule.Version, Operation: rule.Operation, Field: rule.Field,
			Expected: rule.Expected, Number: rule.Number, Values: append([]string(nil), rule.Values...),
		})
	}
	for _, raw := range records {
		if err := ctx.Err(); err != nil {
			return nil, nil, nil, "", domain.ContextFailure("run.transform", err)
		}
		var input map[string]any
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		if err := decoder.Decode(&input); err != nil {
			return nil, nil, nil, "", domain.NewFailure(domain.FailureInvalidInput, "run.transform", "source record cannot be decoded")
		}
		transformed, err := transform.Apply(input, transformRules)
		if err != nil {
			return nil, nil, nil, "", err
		}
		output, err := metadata.CanonicalJSON(transformed.Record)
		if err != nil {
			return nil, nil, nil, "", domain.NewFailure(domain.FailureInvalidInput, "run.transform", "transformed record cannot be canonicalized")
		}
		outputs = append(outputs, output)
		provenance = append(provenance, transformed.Provenance)
		if len(qualityRules) > 0 {
			result, err := quality.Evaluate(transformed.Record, qualityRules, object.QualityFailureDisposition)
			if err != nil {
				return nil, nil, nil, "", err
			}
			results = append(results, result)
		}
	}
	disposition := "pass"
	if len(qualityRules) > 0 {
		disposition = object.QualityFailureDisposition
		for _, result := range results {
			if !result.Passed {
				switch disposition {
				case "warning", "quarantine":
				case "reject", "hold", "cleaning_required":
					return nil, nil, nil, "", domain.NewFailure(domain.FailurePolicyRejected, "run.quality", "quality disposition prevents publication; no destination data was committed")
				}
			}
		}
	}
	return outputs, provenance, results, disposition, nil
}

func appendSynced[T any](path string, value T) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	if !isPrivateFile(path) {
		_ = file.Close()
		return fmt.Errorf("journal file permissions are not private")
	}
	n, writeErr := file.Write(data)
	if writeErr == nil && n != len(data) {
		writeErr = io.ErrShortWrite
	}
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return syncDirectory(filepath.Dir(path))
}

func ensureAudit(path string, event auditEvent) error {
	raw, err := json.Marshal(event)
	if err != nil {
		return domain.NewFailure(domain.FailureInternal, "audit.append", "audit event cannot be encoded")
	}
	return (fileAuditStore{path: path}).Append(context.Background(), raw)
}

func appendAuditEvent(path string, event auditEvent) error {
	if err := ensurePrivateDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	events, err := readAndRepairTail[auditEvent](path)
	if err != nil {
		return err
	}
	for _, prior := range events {
		if prior.EventID == event.EventID {
			if prior.Operation != event.Operation || prior.RunID != event.RunID || prior.MetadataRevision != event.MetadataRevision ||
				prior.CorrectionID != event.CorrectionID || prior.CorrectionVersion != event.CorrectionVersion ||
				prior.InputDigest != event.InputDigest || prior.OutputDigest != event.OutputDigest ||
				prior.PlanID != event.PlanID || prior.Attempt != event.Attempt ||
				prior.OriginalRunID != event.OriginalRunID {
				return domain.NewFailure(domain.FailureConflict, "audit.append", "audit event identity conflicts with prior evidence")
			}
			return nil
		}
	}
	return appendSynced(path, event)
}

func recordRunFailure(path, actor, revision, planID, runID string, attempt, count int, failure domain.Failure) error {
	failureStage := failure.Stage
	if failureStage == "" {
		failureStage = failure.Operation
	}
	event := auditEvent{
		EventID:   digest([]byte(strings.Join([]string{"run.failed", runID, fmt.Sprint(attempt), string(failure.Category), failure.Operation}, "\x00"))),
		Operation: "run.failed", FailureCategory: string(failure.Category), FailureStage: failureStage,
		Actor: actor, MetadataRevision: revision, RunID: runID, PlanID: planID, Attempt: attempt,
		RecordCount: count, OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	return ensureAudit(path, event)
}

func recordDenial(stateDir, actor, operation, category string) error {
	if stateDir == "" {
		return fmt.Errorf("audit directory is required")
	}
	if actor != "local-user" {
		actor = "unknown"
	}
	event := auditEvent{
		EventID:   digest([]byte(strings.Join([]string{"denial", operation, actor, category}, "\x00"))),
		Operation: "access.denied", AttemptedOperation: operation, FailureCategory: category, Actor: actor, RunID: "",
		OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	if category != "access_denied" {
		return fmt.Errorf("unsupported denial category")
	}
	return ensureAudit(filepath.Join(stateDir, "audit", "events.jsonl"), event)
}

func mustJSON(value any) []byte {
	data, _ := json.Marshal(value)
	return data
}

func readAndRepairTail[T any](path string) ([]T, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, domain.MapAdapterFailure(domain.FailureUnavailable, "journal.open", "journal is unavailable", err)
	}
	defer file.Close()
	if !isPrivateFile(path) {
		return nil, domain.NewFailure(domain.FailureAccessDenied, "journal.open", "journal permissions must be private to the local user")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxSourceBytes*2+1))
	if err != nil || len(data) > maxSourceBytes*2 {
		return nil, domain.NewFailure(domain.FailureInvalidInput, "journal.read", "journal exceeds the supported size")
	}
	if len(data) == 0 {
		return []T{}, nil
	}
	lastNewline := bytes.LastIndexByte(data, '\n')
	if lastNewline != len(data)-1 {
		validLength := lastNewline + 1
		if err := file.Truncate(int64(validLength)); err != nil {
			return nil, domain.NewFailure(domain.FailureUnavailable, "journal.repair", "incomplete journal tail cannot be safely truncated")
		}
		data = data[:validLength]
		if err := file.Sync(); err != nil {
			return nil, domain.NewFailure(domain.FailureUnavailable, "journal.repair", "journal tail repair could not be synced")
		}
	}
	lines := bytes.Split(data, []byte{'\n'})
	values := make([]T, 0, len(lines)-1)
	for _, line := range lines[:len(lines)-1] {
		if len(line) == 0 {
			continue
		}
		if err := metadata.ValidateJSONDocument(line); err != nil {
			return nil, domain.NewFailure(domain.FailureInvalidInput, "journal.read", "complete journal entry is malformed; preserve data for operator recovery")
		}
		var value T
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&value); err != nil {
			return nil, domain.NewFailure(domain.FailureInvalidInput, "journal.read", "complete journal entry is malformed; preserve data for operator recovery")
		}
		values = append(values, value)
	}
	return values, nil
}

type lockFile struct {
	path string
	file *os.File
}

func acquireLock(directory string) (*lockFile, error) {
	path := filepath.Join(directory, "LOCK")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, domain.NewFailure(domain.FailureConflict, "run.ownership", "another local process owns this state directory")
		}
		return nil, domain.NewFailure(domain.FailureUnavailable, "run.ownership", "cannot acquire local run ownership")
	}
	if _, err := fmt.Fprintf(file, "pid=%d\n", os.Getpid()); err != nil {
		file.Close()
		os.Remove(path)
		return nil, domain.NewFailure(domain.FailureUnavailable, "run.ownership", "cannot persist local run ownership")
	}
	if err := file.Sync(); err != nil {
		file.Close()
		os.Remove(path)
		return nil, domain.NewFailure(domain.FailureUnavailable, "run.ownership", "cannot sync local run ownership")
	}
	return &lockFile{path: path, file: file}, nil
}

func releaseLock(lock *lockFile) {
	_ = lock.file.Close()
	_ = os.Remove(lock.path)
}

func ReadCommitted(path string) ([]committedBatch, error) {
	return readAndRepairTail[committedBatch](path)
}
