package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/metadata"
)

const (
	ContractVersion                = "mimir.schedule-request/v1"
	AcknowledgementContractVersion = "mimir.schedule-request-ack/v1"
)

type Request struct {
	ContractVersion  string `json:"contractVersion"`
	RequestID        string `json:"requestId"`
	MetadataRevision string `json:"metadataRevision"`
	SourceID         string `json:"sourceId"`
	ObjectID         string `json:"objectId"`
	ScheduledAt      string `json:"scheduledAt"`
	RetryLimit       int    `json:"retryLimit"`
	ConcurrencyLimit int    `json:"concurrencyLimit"`
}

type Acknowledgement struct {
	ContractVersion string `json:"contractVersion"`
	RequestID       string `json:"requestId"`
	State           string `json:"state"`
	Reason          string `json:"reason"`
	AcknowledgedAt  string `json:"acknowledgedAt"`
}

type Decision struct {
	SourceID string   `json:"sourceId"`
	ObjectID string   `json:"objectId"`
	Eligible bool     `json:"eligible"`
	Reason   string   `json:"reason"`
	Request  *Request `json:"request,omitempty"`
}

// Evaluate produces one deterministic request for the current interval slot.
// It performs no ingestion or state transition.
func Evaluate(revision, sourceID, objectID string, schedule metadata.Schedule, now time.Time, completed map[string]bool) (Decision, error) {
	decision := Decision{SourceID: sourceID, ObjectID: objectID}
	if revision == "" || sourceID == "" || objectID == "" || now.IsZero() || schedule.IntervalSeconds < 1 {
		return decision, domain.NewFailure(domain.FailureInvalidInput, "schedule.evaluate", "revision, scope, clock, and positive interval are required")
	}
	if !schedule.Enabled {
		decision.Reason = "disabled"
		return decision, nil
	}
	start, err := parseTime(schedule.StartsAt)
	if err != nil {
		return decision, domain.NewFailure(domain.FailureInvalidInput, "schedule.evaluate", "schedule start time is invalid")
	}
	end, err := parseTime(schedule.EndsAt)
	if err != nil || (!start.IsZero() && !end.IsZero() && !end.After(start)) {
		return decision, domain.NewFailure(domain.FailureInvalidInput, "schedule.evaluate", "schedule window is invalid")
	}
	clock := now.UTC()
	if (!start.IsZero() && clock.Before(start)) || (!end.IsZero() && clock.After(end)) {
		decision.Reason = "outside_window"
		return decision, nil
	}
	for _, dependency := range schedule.Dependencies {
		if !completed[dependency] {
			decision.Reason = "dependency_not_satisfied"
			return decision, nil
		}
	}
	slot := clock.Truncate(time.Duration(schedule.IntervalSeconds) * time.Second)
	if !start.IsZero() && slot.Before(start) {
		slot = start
	}
	concurrencyLimit := schedule.ConcurrencyLimit
	if concurrencyLimit == 0 {
		concurrencyLimit = 1
	}
	request := Request{
		ContractVersion: ContractVersion, MetadataRevision: revision, SourceID: sourceID, ObjectID: objectID,
		ScheduledAt: slot.Format(time.RFC3339), RetryLimit: schedule.RetryLimit, ConcurrencyLimit: concurrencyLimit,
	}
	request.RequestID = metadata.Digest([]byte(fmt.Sprintf("%s\x00%s\x00%s\x00%s", revision, sourceID, objectID, request.ScheduledAt)))
	decision.Eligible = true
	decision.Reason = "eligible"
	decision.Request = &request
	return decision, nil
}

type Dispatcher struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func (d *Dispatcher) Dispatch(request Request) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.seen == nil {
		d.seen = make(map[string]struct{})
	}
	if request.ContractVersion != ContractVersion || request.RequestID == "" {
		return false
	}
	if _, ok := d.seen[request.RequestID]; ok {
		return false
	}
	d.seen[request.RequestID] = struct{}{}
	return true
}

func parseTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid schedule time: %w", err)
	}
	return parsed, nil
}
