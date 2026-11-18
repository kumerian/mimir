package scheduler

import (
	"testing"
	"time"

	"github.com/kumerian/mimir/internal/metadata"
)

func TestEvaluateEmitsStableRequestsOnlyWhenWindowAndDependenciesAllow(t *testing.T) {
	now := time.Date(2026, 11, 18, 10, 4, 17, 0, time.UTC)
	spec := metadata.Schedule{
		Enabled: true, IntervalSeconds: 60, StartsAt: "2026-11-18T10:00:00Z",
		Dependencies: []string{"upstream"}, RetryLimit: 2, ConcurrencyLimit: 1,
	}
	blocked, err := Evaluate("rev-1", "source", "object", spec, now, nil)
	if err != nil || blocked.Eligible || blocked.Reason != "dependency_not_satisfied" {
		t.Fatalf("blocked decision=%+v err=%v", blocked, err)
	}
	first, err := Evaluate("rev-1", "source", "object", spec, now, map[string]bool{"upstream": true})
	if err != nil || !first.Eligible || first.Request == nil {
		t.Fatalf("eligible decision=%+v err=%v", first, err)
	}
	second, err := Evaluate("rev-1", "source", "object", spec, now.Add(20*time.Second), map[string]bool{"upstream": true})
	if err != nil || second.Request.RequestID != first.Request.RequestID {
		t.Fatalf("same interval produced different request: first=%+v second=%+v err=%v", first, second, err)
	}
	dispatcher := &Dispatcher{}
	if !dispatcher.Dispatch(*first.Request) || dispatcher.Dispatch(*second.Request) {
		t.Fatal("duplicate trigger delivery was not deduplicated")
	}
}

func TestEvaluateRejectsInvalidWindowAndReportsDisabledSchedule(t *testing.T) {
	now := time.Date(2026, 11, 18, 10, 4, 17, 0, time.UTC)
	disabled, err := Evaluate("rev", "source", "object", metadata.Schedule{IntervalSeconds: 60}, now, nil)
	if err != nil || disabled.Eligible || disabled.Reason != "disabled" {
		t.Fatalf("disabled decision=%+v err=%v", disabled, err)
	}
	_, err = Evaluate("rev", "source", "object", metadata.Schedule{
		Enabled: true, IntervalSeconds: 60, StartsAt: "not-a-time",
	}, now, nil)
	if err == nil {
		t.Fatal("invalid activation window was accepted")
	}
}
