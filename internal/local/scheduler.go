package local

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/metadata"
	"github.com/kumerian/mimir/internal/scheduler"
	"github.com/kumerian/mimir/internal/state"
)

type ScheduleOptions struct {
	StateDir   string
	Actor      string
	AllowLocal bool
	At         time.Time
}

type ScheduleResult struct {
	Decisions []scheduler.Decision `json:"decisions"`
	Requests  []scheduler.Request  `json:"requests"`
}

type persistedRequest struct {
	Request scheduler.Request `json:"request"`
}

// EvaluateSchedules persists stable trigger requests but never starts a run.
func EvaluateSchedules(ctx context.Context, options ScheduleOptions) (ScheduleResult, error) {
	return evaluateSchedules(ctx, options, nil)
}

func evaluateSchedules(ctx context.Context, options ScheduleOptions, afterRequest func()) (ScheduleResult, error) {
	if err := ctx.Err(); err != nil {
		if options.StateDir != "" && options.Actor == "local-user" && options.AllowLocal {
			if stopErr := StopPendingScheduleRequests(context.WithoutCancel(ctx), options.StateDir, options.Actor, true, "cancelled"); stopErr != nil {
				return ScheduleResult{}, stopErr
			}
		}
		return ScheduleResult{}, domain.ContextFailure("schedule.evaluate", err)
	}
	if !options.AllowLocal || options.Actor != "local-user" {
		if err := recordDenial(options.StateDir, options.Actor, "schedule.evaluate", "access_denied"); err != nil {
			return ScheduleResult{}, domain.NewFailure(domain.FailureUnavailable, "audit.denial", "schedule evaluation was denied and denial audit could not be persisted")
		}
		return ScheduleResult{}, domain.NewFailure(domain.FailureAccessDenied, "schedule.evaluate", "explicit local-user authorization is required")
	}
	info, err := os.Stat(options.StateDir)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return ScheduleResult{}, domain.NewFailure(domain.FailureInvalidInput, "schedule.evaluate", "private local metadata directory is unavailable")
	}
	lock, err := acquireLock(options.StateDir)
	if err != nil {
		return ScheduleResult{}, err
	}
	defer releaseLock(lock)
	parsed, _, err := loadActiveMetadata(options.StateDir)
	if err != nil {
		return ScheduleResult{}, err
	}
	now := options.At
	if now.IsZero() {
		now = time.Now().UTC()
	}
	completed, err := completedObjectIDs(options.StateDir)
	if err != nil {
		return ScheduleResult{}, err
	}
	result := ScheduleResult{Decisions: make([]scheduler.Decision, 0), Requests: make([]scheduler.Request, 0)}
	for _, source := range parsed.Document.Sources {
		for _, object := range source.Objects {
			if err := ctx.Err(); err != nil {
				if stopErr := stopPendingScheduleRequestsLocked(context.WithoutCancel(ctx), options.StateDir, options.Actor, "cancelled"); stopErr != nil {
					return ScheduleResult{}, stopErr
				}
				return ScheduleResult{}, domain.ContextFailure("schedule.evaluate", err)
			}
			if object.Schedule == nil {
				continue
			}
			decision, err := scheduler.Evaluate(parsed.Document.Revision, source.ID, object.ID, *object.Schedule, now, completed)
			if err != nil {
				return ScheduleResult{}, err
			}
			result.Decisions = append(result.Decisions, decision)
			if decision.Request == nil {
				continue
			}
			request := *decision.Request
			path := filepath.Join(options.StateDir, "state", "schedule-requests.jsonl")
			if err := ensurePrivateDirectory(filepath.Dir(path)); err != nil {
				return ScheduleResult{}, domain.NewFailure(domain.FailureUnavailable, "schedule.persist", "schedule request directory is unavailable")
			}
			prior, err := readAndRepairTail[persistedRequest](path)
			if err != nil {
				return ScheduleResult{}, err
			}
			exists := false
			count := 0
			for _, entry := range prior {
				if entry.Request.RequestID == request.RequestID {
					count++
					if entry.Request != request {
						return ScheduleResult{}, domain.NewFailure(domain.FailureConflict, "schedule.persist", "request identity conflicts with persisted schedule")
					}
					exists = true
				}
			}
			if count > 1 {
				return ScheduleResult{}, domain.NewFailure(domain.FailureConflict, "schedule.persist", "multiple requests exist for the same schedule identity")
			}
			if !exists {
				if err := appendSynced(path, persistedRequest{Request: request}); err != nil {
					return ScheduleResult{}, domain.NewFailure(domain.FailureUnavailable, "schedule.persist", "schedule request could not be durably stored")
				}
			}
			event := auditEvent{
				EventID: request.RequestID, Operation: "schedule.requested", Actor: options.Actor,
				MetadataRevision: request.MetadataRevision, PlanID: request.RequestID,
				OccurredAt: now.UTC().Format(time.RFC3339Nano),
			}
			if err := ensureAudit(filepath.Join(options.StateDir, "audit", "events.jsonl"), event); err != nil {
				return ScheduleResult{}, domain.NewFailure(domain.FailureUnavailable, "schedule.audit", "request is stored but its audit event is pending")
			}
			stopped, err := scheduleRequestStopped(options.StateDir, request.RequestID)
			if err != nil {
				return ScheduleResult{}, err
			}
			if stopped {
				continue
			}
			result.Requests = append(result.Requests, request)
			if afterRequest != nil {
				afterRequest()
			}
		}
	}
	if err := ctx.Err(); err != nil {
		if stopErr := stopPendingScheduleRequestsLocked(context.WithoutCancel(ctx), options.StateDir, options.Actor, "cancelled"); stopErr != nil {
			return ScheduleResult{}, stopErr
		}
		return ScheduleResult{}, domain.ContextFailure("schedule.evaluate", err)
	}
	sort.Slice(result.Requests, func(i, j int) bool { return result.Requests[i].RequestID < result.Requests[j].RequestID })
	return result, nil
}

// StopPendingScheduleRequests records a durable stop acknowledgement for every
// request not previously acknowledged. It never creates or changes run state.
func StopPendingScheduleRequests(ctx context.Context, stateDir, actor string, allowLocal bool, reason string) error {
	if err := ctx.Err(); err != nil {
		return domain.ContextFailure("schedule.stop", err)
	}
	if !allowLocal || actor != "local-user" {
		if err := recordDenial(stateDir, actor, "schedule.stop", "access_denied"); err != nil {
			return domain.NewFailure(domain.FailureUnavailable, "audit.denial", "schedule stop was denied and denial audit could not be persisted")
		}
		return domain.NewFailure(domain.FailureAccessDenied, "schedule.stop", "explicit local-user authorization is required")
	}
	if reason != "cancelled" && reason != "shutdown" {
		return domain.NewFailure(domain.FailureInvalidInput, "schedule.stop", "stop reason must be cancelled or shutdown")
	}
	info, err := os.Stat(stateDir)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return domain.NewFailure(domain.FailureInvalidInput, "schedule.stop", "private local metadata directory is unavailable")
	}
	lock, err := acquireLock(stateDir)
	if err != nil {
		return err
	}
	defer releaseLock(lock)
	return stopPendingScheduleRequestsLocked(ctx, stateDir, actor, reason)
}

func stopPendingScheduleRequestsLocked(ctx context.Context, stateDir, actor, reason string) error {
	if err := ctx.Err(); err != nil {
		return domain.ContextFailure("schedule.stop", err)
	}
	requestPath := filepath.Join(stateDir, "state", "schedule-requests.jsonl")
	if _, err := os.Stat(requestPath); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return domain.NewFailure(domain.FailureUnavailable, "schedule.stop", "pending schedule requests are unavailable")
	}
	requests, err := readAndRepairTail[persistedRequest](requestPath)
	if err != nil {
		return err
	}
	ackPath := filepath.Join(stateDir, "state", "schedule-request-acks.jsonl")
	acks, err := readAndRepairTail[scheduler.Acknowledgement](ackPath)
	if err != nil {
		return err
	}
	acknowledged := make(map[string]scheduler.Acknowledgement, len(acks))
	for _, acknowledgement := range acks {
		if acknowledgement.ContractVersion != scheduler.AcknowledgementContractVersion ||
			acknowledgement.RequestID == "" || acknowledgement.State != "stopped" ||
			(acknowledgement.Reason != "cancelled" && acknowledgement.Reason != "shutdown") {
			return domain.NewFailure(domain.FailureInvalidInput, "schedule.stop", "stored request acknowledgement has an unsupported contract")
		}
		if _, exists := acknowledged[acknowledgement.RequestID]; exists {
			return domain.NewFailure(domain.FailureConflict, "schedule.stop", "duplicate request acknowledgement exists")
		}
		acknowledged[acknowledgement.RequestID] = acknowledgement
	}
	seen := make(map[string]bool, len(requests))
	for _, entry := range requests {
		request := entry.Request
		if request.ContractVersion != scheduler.ContractVersion || request.RequestID == "" ||
			seen[request.RequestID] {
			return domain.NewFailure(domain.FailureInvalidInput, "schedule.stop", "stored schedule request is invalid")
		}
		seen[request.RequestID] = true
		acknowledgement, exists := acknowledged[request.RequestID]
		if !exists {
			acknowledgement = scheduler.Acknowledgement{
				ContractVersion: scheduler.AcknowledgementContractVersion, RequestID: request.RequestID,
				State: "stopped", Reason: reason, AcknowledgedAt: time.Now().UTC().Format(time.RFC3339Nano),
			}
			if err := appendSynced(ackPath, acknowledgement); err != nil {
				return domain.NewFailure(domain.FailureUnavailable, "schedule.stop", "pending request stop acknowledgement could not be persisted")
			}
			acknowledged[request.RequestID] = acknowledgement
		}
		event := auditEvent{
			EventID:   metadata.Digest([]byte("schedule.stopped\x00" + request.RequestID)),
			Operation: "schedule.stopped", Actor: actor, MetadataRevision: request.MetadataRevision,
			PlanID: request.RequestID, OccurredAt: acknowledgement.AcknowledgedAt,
		}
		if err := ensureAudit(filepath.Join(stateDir, "audit", "events.jsonl"), event); err != nil {
			return domain.NewFailure(domain.FailureUnavailable, "schedule.stop", "stop acknowledgement is stored but its audit event is pending")
		}
	}
	return nil
}

func scheduleRequestStopped(stateDir, requestID string) (bool, error) {
	acks, err := readAndRepairTail[scheduler.Acknowledgement](filepath.Join(stateDir, "state", "schedule-request-acks.jsonl"))
	if err != nil {
		return false, err
	}
	for _, acknowledgement := range acks {
		if acknowledgement.RequestID == requestID {
			if acknowledgement.ContractVersion != scheduler.AcknowledgementContractVersion || acknowledgement.State != "stopped" {
				return false, domain.NewFailure(domain.FailureInvalidInput, "schedule.request", "request acknowledgement has an unsupported contract")
			}
			return true, nil
		}
	}
	return false, nil
}

func completedObjectIDs(stateDir string) (map[string]bool, error) {
	completed := make(map[string]bool)
	audits, err := readAndRepairTail[auditEvent](filepath.Join(stateDir, "audit", "events.jsonl"))
	if err != nil {
		return nil, err
	}
	statePath := filepath.Join(stateDir, "state", "runs.jsonl")
	if err := ensurePrivateDirectory(filepath.Dir(statePath)); err != nil {
		return nil, domain.NewFailure(domain.FailureUnavailable, "schedule.dependencies", "run-state directory is unavailable")
	}
	states, err := readAndRepairTail[state.Event](statePath)
	if err != nil {
		return nil, err
	}
	succeeded := make(map[string]bool)
	for _, event := range states {
		succeeded[event.RunID] = event.Status == state.Succeeded
	}
	for _, event := range audits {
		if event.Operation != "run.committed" || event.RunID == "" {
			continue
		}
		if !succeeded[event.RunID] {
			continue
		}
		var plan persistedPlan
		if err := readStrictJSON(filepath.Join(stateDir, "plans", event.RunID+".json"), &plan); err != nil {
			return nil, domain.NewFailure(domain.FailureInvalidInput, "schedule.dependencies", "prior run plan cannot be verified")
		}
		completed[plan.Plan.ObjectID] = true
	}
	return completed, nil
}
