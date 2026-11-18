package state

import (
	"testing"
	"time"

	"github.com/kumerian/mimir/internal/domain"
)

func TestRunStateFencesOwnersAndPreservesRetryHistory(t *testing.T) {
	now := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)
	first, created, err := StartAttempt(nil, "run", "plan", "owner-1", now)
	if err != nil || !created || first.Attempt != 1 {
		t.Fatalf("first attempt=%+v created=%v err=%v", first, created, err)
	}
	if _, err := Transition(first, "stale-owner", Succeeded, "", now); err == nil {
		t.Fatal("stale owner was allowed to complete a run")
	}
	failed, err := Transition(first, "owner-1", Failed, domain.FailureUnavailable, now)
	if err != nil {
		t.Fatal(err)
	}
	second, created, err := StartAttempt([]Event{first, failed}, "run", "plan", "owner-2", now.Add(time.Second))
	if err != nil || !created || second.Attempt != 2 {
		t.Fatalf("retry attempt=%+v created=%v err=%v", second, created, err)
	}
	if second.FailureCategory != "" || failed.FailureCategory != domain.FailureUnavailable {
		t.Fatal("retry overwrote prior failure evidence")
	}
}

func TestRunStateRejectsInvalidTransitionsAndPlanChanges(t *testing.T) {
	now := time.Now().UTC()
	first, _, err := StartAttempt(nil, "run", "plan", "owner", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Transition(first, "owner", Status("planned"), "", now); err == nil {
		t.Fatal("unsupported transition was accepted")
	}
	if _, _, err := StartAttempt([]Event{first}, "run", "other-plan", "owner-2", now); err == nil {
		t.Fatal("state was rebound to a different immutable plan")
	}
}
