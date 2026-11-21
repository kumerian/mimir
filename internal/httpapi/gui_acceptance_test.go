//go:build acceptance

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kumerian/mimir/internal/local"
	"github.com/kumerian/mimir/internal/metadata"
)

type guiAcceptanceOperations struct {
	Operations
}

func (guiAcceptanceOperations) Health(context.Context) (ProfileHealth, error) {
	return ProfileHealth{
		Status:    "ready",
		Profile:   "local-reference-v1",
		Contracts: []string{ContractVersion},
	}, nil
}

func (guiAcceptanceOperations) Validate(context.Context, json.RawMessage) (metadata.ValidationReport, error) {
	return metadata.ValidationReport{
		ContractVersion:       metadata.ValidationReportVersion,
		InputDigest:           strings.Repeat("a", 64),
		Valid:                 true,
		RunEligible:           false,
		SecurityImpact:        "input_not_retained",
		CompatibilityDecision: "accepted_v1",
	}, nil
}

func (guiAcceptanceOperations) Author(context.Context, json.RawMessage) (local.RevisionSummary, error) {
	return local.RevisionSummary{Revision: "synthetic-orders-v1", Status: "draft", Digest: strings.Repeat("b", 64)}, nil
}

func (guiAcceptanceOperations) Activate(context.Context, ActivateRequest) (local.ActivationResult, error) {
	return local.ActivationResult{Revision: "synthetic-orders-v1", Status: "active", Digest: strings.Repeat("b", 64)}, nil
}

func (guiAcceptanceOperations) Discover(context.Context, DiscoveryRequest) (local.DiscoveryResult, error) {
	return local.DiscoveryResult{ProposalID: "synthetic-proposal"}, nil
}

func (guiAcceptanceOperations) Run(context.Context, RunRequest) (local.Result, error) {
	return local.Result{
		RunID:            "c",
		MetadataRevision: "synthetic-orders-v1",
		RecordCount:      1,
		AlreadyCommitted: false,
	}, nil
}

func (guiAcceptanceOperations) EvaluateSchedules(context.Context, ScheduleRequest) (local.ScheduleResult, error) {
	return local.ScheduleResult{}, nil
}

func (guiAcceptanceOperations) StopScheduleRequests(context.Context, StopScheduleRequest) error {
	return nil
}

func (guiAcceptanceOperations) Replay(context.Context, ReplayRequest) (local.ReplayResult, error) {
	return local.ReplayResult{
		ReplayPlanID:  "synthetic-replay",
		OriginalRunID: "d",
		Result: local.Result{
			RunID:            "d",
			MetadataRevision: "synthetic-orders-v1",
			RecordCount:      1,
			AlreadyCommitted: true,
		},
	}, nil
}

func (guiAcceptanceOperations) Clean(context.Context, CleaningRequest, bool) (local.CleaningResult, error) {
	return local.CleaningResult{
		Record:  map[string]any{"name": "Example"},
		Preview: true,
	}, nil
}

func guiAcceptanceRequest(method, target, body string) *http.Request {
	request := httptest.NewRequest(method, "http://127.0.0.1:8787"+target, strings.NewReader(body))
	request.Host = "127.0.0.1:8787"
	request.Header.Set("Mimir-Contract-Version", ContractVersion)
	request.Header.Set("X-Correlation-ID", "gui-acceptance")
	if method == http.MethodPost {
		request.Header.Set("Origin", "http://127.0.0.1:8787")
		request.Header.Set("Content-Type", "application/json")
	}
	return request
}

func TestGUICandidateSyntheticAcceptance(t *testing.T) {
	handler := NewHandler(guiAcceptanceOperations{})

	checks := []struct {
		name   string
		method string
		target string
		body   string
		status int
	}{
		{"health identity only", http.MethodGet, "/api/v1/health", "", http.StatusOK},
		{"validate metadata", http.MethodPost, "/api/v1/metadata/validate", `{"schemaVersion":"mimir.metadata/v1"}`, http.StatusOK},
		{"author draft", http.MethodPost, "/api/v1/metadata/revisions", `{"schemaVersion":"mimir.metadata/v1","revision":"synthetic-orders-v1","status":"draft","sources":[]}`, http.StatusCreated},
		{"activate draft", http.MethodPost, "/api/v1/metadata/activate", `{"revision":"synthetic-orders-v1","expectedActive":"old-v1"}`, http.StatusOK},
		{"advisory discovery", http.MethodPost, "/api/v1/discovery", `{"sourceRoot":"/private/source","disposition":"continue"}`, http.StatusOK},
		{"bounded run", http.MethodPost, "/api/v1/runs", `{"sourceRoot":"/private/source"}`, http.StatusOK},
		{"schedule evaluation", http.MethodPost, "/api/v1/schedules/evaluate", `{}`, http.StatusOK},
		{"stop pending requests", http.MethodPost, "/api/v1/schedules/stop", `{"reason":"shutdown"}`, http.StatusOK},
		{"constrained replay", http.MethodPost, "/api/v1/replays", `{"sourceRoot":"/private/source","runId":"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd","reason":"recovery"}`, http.StatusOK},
		{"cleaning preview", http.MethodPost, "/api/v1/cleaning/preview", `{"record":{"name":" Example "},"classification":"internal","correction":{"id":"trim-name","version":"v1","operation":"trim","field":"name"}}`, http.StatusOK},
		{"cleaning apply", http.MethodPost, "/api/v1/cleaning/apply", `{"record":{"name":" Example "},"classification":"internal","correction":{"id":"trim-name","version":"v1","operation":"trim","field":"name"}}`, http.StatusOK},
		{"unsupported query absent", http.MethodGet, "/api/v1/runs/history", "", http.StatusBadRequest},
	}

	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, guiAcceptanceRequest(check.method, check.target, check.body))
			if response.Code != check.status {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, check.status, response.Body.String())
			}
		})
	}
}
