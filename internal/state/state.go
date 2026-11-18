package state

import (
	"fmt"
	"time"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/metadata"
)

const ContractVersion = "mimir.run-state/v1"

type Status string

const (
	Running   Status = "running"
	Succeeded Status = "succeeded"
	Failed    Status = "failed"
	Cancelled Status = "cancelled"
)

type Event struct {
	ContractVersion string                 `json:"contractVersion"`
	RunID           string                 `json:"runId"`
	PlanID          string                 `json:"planId"`
	Attempt         int                    `json:"attempt"`
	Status          Status                 `json:"status"`
	OwnerToken      string                 `json:"ownerToken"`
	FailureCategory domain.FailureCategory `json:"failureCategory,omitempty"`
	FailureStage    string                 `json:"failureStage,omitempty"`
	OccurredAt      string                 `json:"occurredAt"`
}

// StartAttempt preserves prior events and allocates a new owner-fenced attempt.
func StartAttempt(events []Event, runID, planID, owner string, now time.Time) (Event, bool, error) {
	if _, err := domain.NewAttempt(runID, planID, 1); err != nil || owner == "" || now.IsZero() {
		return Event{}, false, domain.NewFailure(domain.FailureInvalidInput, "run.state.start", "run, plan, owner, and clock are required")
	}
	lastAttempt := 0
	for _, event := range events {
		if event.RunID != runID {
			continue
		}
		if event.ContractVersion != ContractVersion {
			return Event{}, false, domain.NewFailure(domain.FailureUnsupportedCapability, "run.state.start", "persisted run-state version is unsupported")
		}
		if _, err := domain.NewAttempt(event.RunID, event.PlanID, event.Attempt); err != nil ||
			event.PlanID != planID || event.OwnerToken == "" {
			return Event{}, false, domain.NewFailure(domain.FailureConflict, "run.state.start", "run history conflicts with the immutable plan")
		}
		if event.Attempt > lastAttempt {
			lastAttempt = event.Attempt
		}
	}
	if lastAttempt == 0 {
		return Event{
			ContractVersion: ContractVersion, RunID: runID, PlanID: planID, Attempt: 1,
			Status: Running, OwnerToken: owner, OccurredAt: now.UTC().Format(time.RFC3339Nano),
		}, true, nil
	}
	last := latestAttempt(events, runID, lastAttempt)
	if last.Status == Succeeded {
		return last, false, nil
	}
	if last.Status != Running && last.Status != Failed && last.Status != Cancelled {
		return Event{}, false, domain.NewFailure(domain.FailureConflict, "run.state.start", "run history has an unsupported terminal state")
	}
	return Event{
		ContractVersion: ContractVersion, RunID: runID, PlanID: planID, Attempt: lastAttempt + 1,
		Status: Running, OwnerToken: owner, OccurredAt: now.UTC().Format(time.RFC3339Nano),
	}, true, nil
}

func Transition(current Event, owner string, next Status, category domain.FailureCategory, now time.Time, failureStage ...string) (Event, error) {
	if _, err := domain.NewAttempt(current.RunID, current.PlanID, current.Attempt); err != nil ||
		current.ContractVersion != ContractVersion || current.Status != Running || current.OwnerToken == "" || current.OwnerToken != owner {
		return Event{}, domain.NewFailure(domain.FailureConflict, "run.state.transition", "stale or invalid run owner cannot change state")
	}
	if now.IsZero() {
		return Event{}, domain.NewFailure(domain.FailureInvalidInput, "run.state.transition", "transition clock is required")
	}
	switch next {
	case Succeeded:
		if category != "" || len(failureStage) != 0 {
			return Event{}, domain.NewFailure(domain.FailureInvalidInput, "run.state.transition", "successful state cannot include a failure category")
		}
	case Failed, Cancelled:
		if category == "" || len(failureStage) > 1 {
			return Event{}, domain.NewFailure(domain.FailureInvalidInput, "run.state.transition", "failed state requires a stable failure category")
		}
	default:
		return Event{}, domain.NewFailure(domain.FailureInvalidInput, "run.state.transition", fmt.Sprintf("unsupported run state %q", next))
	}
	current.Status = next
	current.FailureCategory = category
	current.FailureStage = ""
	if len(failureStage) == 1 {
		current.FailureStage = failureStage[0]
	}
	current.OccurredAt = now.UTC().Format(time.RFC3339Nano)
	return current, nil
}

func latestAttempt(events []Event, runID string, attempt int) Event {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].RunID == runID && events[i].Attempt == attempt {
			return events[i]
		}
	}
	return Event{}
}

func RequestID(revision, scope, triggerSlot string) string {
	return metadata.Digest([]byte(revision + "\x00" + scope + "\x00" + triggerSlot))
}
