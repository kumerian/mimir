package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/local"
	"github.com/kumerian/mimir/internal/metadata"
)

type operationSpy struct {
	Operations
	validateCalls int
	cleanCalls    int
	runCalls      int
	runContext    context.Context
	healthErr     error
	mu            sync.Mutex
}

func (s *operationSpy) Validate(_ context.Context, _ json.RawMessage) (metadata.ValidationReport, error) {
	s.validateCalls++
	return metadata.ValidationReport{ContractVersion: metadata.ValidationReportVersion, Valid: true}, nil
}

func (s *operationSpy) Health(context.Context) (ProfileHealth, error) {
	if s.healthErr != nil {
		return ProfileHealth{}, s.healthErr
	}
	return ProfileHealth{Status: "ready", Profile: "local-reference-v1"}, nil
}

func (s *operationSpy) Clean(_ context.Context, _ CleaningRequest, _ bool) (local.CleaningResult, error) {
	s.cleanCalls++
	return local.CleaningResult{}, nil
}

func (s *operationSpy) Run(ctx context.Context, _ RunRequest) (local.Result, error) {
	s.mu.Lock()
	s.runCalls++
	s.runContext = ctx
	s.mu.Unlock()
	<-ctx.Done()
	return local.Result{}, domain.ContextFailure("run.start", ctx.Err())
}

func apiRequest(method, target, body string) *http.Request {
	request := httptest.NewRequest(method, "http://127.0.0.1:8787"+target, strings.NewReader(body))
	request.Host = "127.0.0.1:8787"
	request.Header.Set("Mimir-Contract-Version", ContractVersion)
	request.Header.Set("X-Correlation-ID", "test-correlation_1")
	if method == http.MethodPost {
		request.Header.Set("Origin", "http://127.0.0.1:8787")
		request.Header.Set("Content-Type", "application/json")
	}
	return request
}

func TestValidateEndpointInvokesOneExistingUseCase(t *testing.T) {
	spy := &operationSpy{}
	response := httptest.NewRecorder()
	NewHandler(spy).ServeHTTP(response, apiRequest(
		http.MethodPost,
		"/api/v1/metadata/validate",
		`{"schemaVersion":"mimir.metadata/v1"}`,
	))

	if response.Code != http.StatusOK {
		t.Fatalf("validate status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	if spy.validateCalls != 1 {
		t.Fatalf("Validate calls = %d, want exactly one", spy.validateCalls)
	}
	if response.Header().Get("Mimir-Contract-Version") != ContractVersion ||
		response.Header().Get("X-Correlation-ID") != "test-correlation_1" {
		t.Fatalf("contract/correlation headers were not preserved: %v", response.Header())
	}
}

func TestRequestGuardsRejectBeforeCallingUseCases(t *testing.T) {
	oversizedMetadata := `{"value":"` + strings.Repeat("x", metadataLimit) + `"}`
	tests := []struct {
		name       string
		request    func() *http.Request
		wantStatus int
		wantKind   domain.FailureCategory
	}{
		{
			name: "missing API contract version",
			request: func() *http.Request {
				r := apiRequest(http.MethodPost, "/api/v1/metadata/validate", `{}`)
				r.Header.Del("Mimir-Contract-Version")
				return r
			},
			wantStatus: http.StatusBadRequest, wantKind: domain.FailureInvalidInput,
		},
		{
			name: "unsupported API contract version",
			request: func() *http.Request {
				r := apiRequest(http.MethodPost, "/api/v1/metadata/validate", `{}`)
				r.Header.Set("Mimir-Contract-Version", "mimir.gui-api/v2")
				return r
			},
			wantStatus: http.StatusUnprocessableEntity, wantKind: domain.FailureUnsupportedCapability,
		},
		{
			name: "cross-origin browser mutation",
			request: func() *http.Request {
				r := apiRequest(http.MethodPost, "/api/v1/metadata/validate", `{}`)
				r.Header.Set("Origin", "http://evil.example")
				r.Host = "evil.example"
				return r
			},
			wantStatus: http.StatusForbidden, wantKind: domain.FailureAccessDenied,
		},
		{
			name: "mismatched Origin and request Host",
			request: func() *http.Request {
				r := apiRequest(http.MethodPost, "/api/v1/metadata/validate", `{}`)
				r.Host = "localhost:8787"
				return r
			},
			wantStatus: http.StatusForbidden, wantKind: domain.FailureAccessDenied,
		},
		{
			name: "missing Origin",
			request: func() *http.Request {
				r := apiRequest(http.MethodPost, "/api/v1/metadata/validate", `{}`)
				r.Header.Del("Origin")
				return r
			},
			wantStatus: http.StatusForbidden, wantKind: domain.FailureAccessDenied,
		},
		{
			name: "forwarded host cannot override request host",
			request: func() *http.Request {
				r := apiRequest(http.MethodPost, "/api/v1/metadata/validate", `{}`)
				r.Header.Set("X-Forwarded-Host", "127.0.0.1:8787")
				r.Host = "service.example"
				return r
			},
			wantStatus: http.StatusForbidden, wantKind: domain.FailureAccessDenied,
		},
		{
			name: "OPTIONS preflight is unsupported",
			request: func() *http.Request {
				return apiRequest(http.MethodOptions, "/api/v1/metadata/validate", "")
			},
			wantStatus: http.StatusBadRequest, wantKind: domain.FailureInvalidInput,
		},
		{
			name: "wrong content type",
			request: func() *http.Request {
				r := apiRequest(http.MethodPost, "/api/v1/metadata/validate", `{}`)
				r.Header.Set("Content-Type", "text/plain")
				return r
			},
			wantStatus: http.StatusBadRequest, wantKind: domain.FailureInvalidInput,
		},
		{
			name: "metadata over body limit",
			request: func() *http.Request {
				return apiRequest(http.MethodPost, "/api/v1/metadata/validate", oversizedMetadata)
			},
			wantStatus: http.StatusBadRequest, wantKind: domain.FailureInvalidInput,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spy := &operationSpy{}
			response := httptest.NewRecorder()
			NewHandler(spy).ServeHTTP(response, test.request())
			var failure domain.Failure
			if err := json.Unmarshal(response.Body.Bytes(), &failure); err != nil {
				t.Fatalf("failure envelope is invalid: %v: %s", err, response.Body.String())
			}
			if response.Code != test.wantStatus || failure.Category != test.wantKind {
				t.Fatalf("status/category = %d/%s, want %d/%s", response.Code, failure.Category, test.wantStatus, test.wantKind)
			}
			if spy.validateCalls != 0 {
				t.Fatalf("validation use case called %d times for rejected request", spy.validateCalls)
			}
			if response.Header().Get("Access-Control-Allow-Origin") != "" {
				t.Fatal("API unexpectedly enabled CORS")
			}
		})
	}
}

func TestUnknownFieldsAndUnsupportedQueryAreRejectedWithoutSideEffects(t *testing.T) {
	spy := &operationSpy{}
	handler := NewHandler(spy)
	for _, request := range []*http.Request{
		apiRequest(http.MethodPost, "/api/v1/cleaning/preview", `{"recordPath":"/etc/passwd","record":{},"classification":"public","correction":{}}`),
		apiRequest(http.MethodGet, "/api/v1/runs/history", ""),
		apiRequest(http.MethodGet, "/api/v1/runs", ""),
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code < 400 {
			t.Fatalf("unsupported request returned %d: %s", response.Code, response.Body.String())
		}
		var failure domain.Failure
		if err := json.Unmarshal(response.Body.Bytes(), &failure); err != nil || failure.ContractVersion != domain.FailureContractVersion {
			t.Fatalf("unsupported request did not return the stable failure envelope: %v: %s", err, response.Body.String())
		}
	}
	if spy.cleanCalls != 0 || spy.runCalls != 0 {
		t.Fatalf("unsupported requests invoked operations: clean=%d run=%d", spy.cleanCalls, spy.runCalls)
	}
}

func TestCleaningRecordLimitIsCheckedBeforeUseCase(t *testing.T) {
	spy := &operationSpy{}
	record := `{"v":"` + strings.Repeat("x", cleaningLimit) + `"}`
	body := fmt.Sprintf(`{"record":%s,"classification":"public","correction":{"id":"c","version":"v1","operation":"trim","field":"v"}}`, record)
	response := httptest.NewRecorder()
	NewHandler(spy).ServeHTTP(response, apiRequest(http.MethodPost, "/api/v1/cleaning/preview", body))
	if response.Code != http.StatusBadRequest || spy.cleanCalls != 0 {
		t.Fatalf("oversized record status=%d clean calls=%d body=%s", response.Code, spy.cleanCalls, response.Body.String())
	}
}

func TestRequestCancellationReachesExistingRunUseCase(t *testing.T) {
	spy := &operationSpy{}
	ctx, cancel := context.WithCancel(context.Background())
	request := apiRequest(http.MethodPost, "/api/v1/runs", `{"sourceRoot":"/synthetic/source"}`).WithContext(ctx)
	response := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		NewHandler(spy).ServeHTTP(response, request)
		close(done)
	}()

	for {
		spy.mu.Lock()
		called := spy.runCalls == 1
		spy.mu.Unlock()
		if called {
			break
		}
		select {
		case <-done:
			t.Fatal("run handler returned before invoking use case")
		default:
		}
	}
	cancel()
	<-done
	spy.mu.Lock()
	defer spy.mu.Unlock()
	if spy.runCalls != 1 || spy.runContext.Err() != context.Canceled {
		t.Fatalf("run use case context/calls = %v/%d, cancellation was not propagated", spy.runContext.Err(), spy.runCalls)
	}
	var failure domain.Failure
	if err := json.Unmarshal(response.Body.Bytes(), &failure); err != nil || failure.Category != domain.FailureCancelled {
		t.Fatalf("cancellation response=%s err=%v", response.Body.String(), err)
	}
}

func TestFailureMappingPreservesUnknownCommitAndHidesRawErrors(t *testing.T) {
	spy := &operationSpy{healthErr: domain.NewFailure(domain.FailureAmbiguousCommit, "cleaning.persist", "safe uncertain result")}
	response := httptest.NewRecorder()
	NewHandler(spy).ServeHTTP(response, apiRequest(http.MethodGet, "/api/v1/health", ""))
	var failure domain.Failure
	if err := json.Unmarshal(response.Body.Bytes(), &failure); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusInternalServerError ||
		failure.Category != domain.FailureAmbiguousCommit ||
		failure.CommitCertainty != domain.CommitUnknown ||
		failure.Reconciliation != "reconcile_before_retry" ||
		failure.BlindRetrySafe() {
		t.Fatalf("ambiguous failure mapping lost commit uncertainty: %+v status=%d", failure, response.Code)
	}

	spy.healthErr = errors.New("raw-adapter-secret-canary")
	response = httptest.NewRecorder()
	NewHandler(spy).ServeHTTP(response, apiRequest(http.MethodGet, "/api/v1/health", ""))
	if strings.Contains(response.Body.String(), "raw-adapter-secret-canary") {
		t.Fatal("raw adapter error was included in response")
	}
}

func TestLocalOperationsRequireExplicitGrantAndAbsoluteStateDirectory(t *testing.T) {
	profile := ProfileHealth{Status: "ready", Profile: "local-reference-v1"}
	if _, err := NewLocalOperations(t.TempDir(), false, profile); err == nil {
		t.Fatal("local operations started without the explicit grant")
	}
	if _, err := NewLocalOperations("relative-state", true, profile); err == nil {
		t.Fatal("local operations accepted a relative state directory")
	}
	if _, err := NewLocalOperations(t.TempDir(), true, profile); err != nil {
		t.Fatalf("valid local startup config rejected: %v", err)
	}
}

func TestListenAddressMustBeLoopbackIPLiteral(t *testing.T) {
	for _, address := range []string{"127.0.0.1:8787", "[::1]:8787"} {
		if err := ValidateListenAddress(address); err != nil {
			t.Errorf("loopback listen address %q rejected: %v", address, err)
		}
	}
	for _, address := range []string{"0.0.0.0:8787", ":8787", "192.0.2.1:8787", "localhost:8787", "127.0.0.1:0"} {
		if err := ValidateListenAddress(address); err == nil {
			t.Errorf("unsafe or invalid listen address %q was accepted", address)
		}
	}
}

func TestContainerListenerAndDatabaseReadinessAreExplicit(t *testing.T) {
	if err := ValidateListenAddressMode("0.0.0.0:8787", true); err != nil {
		t.Fatalf("explicit container listener rejected: %v", err)
	}
	if err := ValidateListenAddressMode("0.0.0.0:8787", false); err == nil {
		t.Fatal("standalone mode accepted a non-loopback listener")
	}

	handler := NewHandlerWithReadiness(&operationSpy{}, func(context.Context) error {
		return errors.New("database unavailable")
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), "not_ready") {
		t.Fatalf("unavailable metadata database readiness = %d %q", response.Code, response.Body.String())
	}
}

func TestContainerSourceRootsStayWithinReadOnlyMount(t *testing.T) {
	profile := ProfileHealth{Status: "ready", Profile: "local-reference-v1"}
	operations, err := NewContainerLocalOperationsWithStores(
		t.TempDir(), true, profile, nil, nil, nil, "/data/sources",
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{"/data/sources", "/data/sources/nested"} {
		if err := operations.validateSourceRoot(root); err != nil {
			t.Errorf("mounted source root %q rejected: %v", root, err)
		}
	}
	for _, root := range []string{"/etc", "/data/sources/../state", "relative"} {
		if err := operations.validateSourceRoot(root); err == nil {
			t.Errorf("source root outside mounted source directory %q accepted", root)
		}
	}
}

func TestOpenAPIContractContainsOnlyImplementedRoutes(t *testing.T) {
	raw, err := os.ReadFile("../../contracts/openapi/local-profile-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		OpenAPI string                    `json:"openapi"`
		Info    struct{ Version string }  `json:"info"`
		Paths   map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("OpenAPI contract is invalid JSON: %v", err)
	}
	if document.OpenAPI != "3.1.0" || document.Info.Version != ContractVersion {
		t.Fatalf("contract identity = %s/%s", document.OpenAPI, document.Info.Version)
	}
	want := map[string][]string{
		"/health":            {"get"},
		"/metadata/validate": {"post"},
		"/metadata/validation-reports/{inputDigest}": {"get"},
		"/metadata/revisions":                        {"get", "post"},
		"/metadata/activate":                         {"post"},
		"/discovery":                                 {"post"},
		"/runs":                                      {"post"},
		"/schedules/evaluate":                        {"post"},
		"/schedules/stop":                            {"post"},
		"/replays":                                   {"post"},
		"/cleaning/preview":                          {"post"},
		"/cleaning/apply":                            {"post"},
	}
	if len(document.Paths) != len(want) {
		t.Fatalf("OpenAPI path count=%d, want %d", len(document.Paths), len(want))
	}
	for route, methods := range want {
		declared, ok := document.Paths[route]
		if !ok || len(declared) != len(methods) {
			t.Errorf("OpenAPI methods for %s = %v, want %v", route, declared, methods)
			continue
		}
		for _, method := range methods {
			if _, ok := declared[method]; !ok {
				t.Errorf("OpenAPI omits %s %s", method, route)
			}
		}
	}
	if _, ok := document.Paths["/runs/history"]; ok {
		t.Fatal("OpenAPI exposes an unavailable run-history query")
	}
}

func TestErrorStatusMappingCoversStableCategories(t *testing.T) {
	tests := []struct {
		category domain.FailureCategory
		status   int
	}{
		{domain.FailureInvalidInput, http.StatusBadRequest},
		{domain.FailureUnsupportedCapability, http.StatusUnprocessableEntity},
		{domain.FailureAccessDenied, http.StatusForbidden},
		{domain.FailureConflict, http.StatusConflict},
		{domain.FailurePolicyRejected, http.StatusUnprocessableEntity},
		{domain.FailureUnavailable, http.StatusServiceUnavailable},
		{domain.FailureAmbiguousCommit, http.StatusInternalServerError},
		{domain.FailureCancelled, http.StatusRequestTimeout},
		{domain.FailureTimeout, http.StatusGatewayTimeout},
		{domain.FailureInternal, http.StatusInternalServerError},
	}
	for _, test := range tests {
		if got := statusFor(test.category); got != test.status {
			t.Errorf("status for %s = %d, want %d", test.category, got, test.status)
		}
	}
}

func TestLocalProfileHTTPVerticalSliceUsesExistingUseCases(t *testing.T) {
	root := t.TempDir()
	stateDir := filepath.Join(root, "state")
	sourceRoot := filepath.Join(root, "source")
	if err := os.Mkdir(sourceRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	sourceRecord := `{"id":7,"canary":"source-value-not-returned"}`
	if err := os.WriteFile(filepath.Join(sourceRoot, "orders.jsonl"), []byte(sourceRecord+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	document := json.RawMessage(`{"schemaVersion":"mimir.metadata/v1","revision":"http-api-v1","status":"draft","sources":[{"id":"orders","classification":"internal","objects":[{"id":"orders","path":"orders.jsonl"}]}]}`)
	profile := ProfileHealth{Status: "ready", Version: "test", Commit: "synthetic", Contracts: []string{ContractVersion}, Profile: "local-reference-v1"}
	operations, err := NewLocalOperations(stateDir, true, profile)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(operations)
	request := func(method, route string, body []byte) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, apiRequest(method, route, string(body)))
		if recorder.Code >= 500 {
			t.Fatalf("%s %s returned %d: %s", method, route, recorder.Code, recorder.Body.String())
		}
		return recorder
	}
	requireStatus := func(response *httptest.ResponseRecorder, status int, operation string) {
		t.Helper()
		if response.Code != status {
			t.Fatalf("%s status=%d want=%d body=%s", operation, response.Code, status, response.Body.String())
		}
	}

	health := request(http.MethodGet, "/api/v1/health", nil)
	requireStatus(health, http.StatusOK, "health")
	var healthInfo ProfileHealth
	if err := json.Unmarshal(health.Body.Bytes(), &healthInfo); err != nil || !contains(healthInfo.Contracts, ContractVersion) {
		t.Fatalf("API profile identity omitted its contract version: %+v err=%v", healthInfo, err)
	}
	validated := request(http.MethodPost, "/api/v1/metadata/validate", document)
	requireStatus(validated, http.StatusOK, "validate")
	var report metadata.ValidationReport
	if err := json.Unmarshal(validated.Body.Bytes(), &report); err != nil || !report.Valid || report.RunEligible {
		t.Fatalf("validation report=%+v err=%v", report, err)
	}

	lookup := request(http.MethodGet, "/api/v1/metadata/validation-reports/"+report.InputDigest, nil)
	requireStatus(lookup, http.StatusOK, "validation report lookup")

	authored := request(http.MethodPost, "/api/v1/metadata/revisions", document)
	requireStatus(authored, http.StatusCreated, "author")
	var revision local.RevisionSummary
	if err := json.Unmarshal(authored.Body.Bytes(), &revision); err != nil || revision.Revision != "http-api-v1" {
		t.Fatalf("author result=%+v err=%v", revision, err)
	}
	listed := request(http.MethodGet, "/api/v1/metadata/revisions", nil)
	requireStatus(listed, http.StatusOK, "list metadata")
	activated := request(http.MethodPost, "/api/v1/metadata/activate", []byte(`{"revision":"http-api-v1"}`))
	requireStatus(activated, http.StatusOK, "activate")

	discovered := request(http.MethodPost, "/api/v1/discovery", []byte(fmt.Sprintf(`{"sourceRoot":%q}`, sourceRoot)))
	requireStatus(discovered, http.StatusOK, "discover")
	scheduled := request(http.MethodPost, "/api/v1/schedules/evaluate", []byte(`{}`))
	requireStatus(scheduled, http.StatusOK, "schedule evaluation")
	stopped := request(http.MethodPost, "/api/v1/schedules/stop", []byte(`{"reason":"shutdown"}`))
	requireStatus(stopped, http.StatusOK, "schedule stop")

	run := request(http.MethodPost, "/api/v1/runs", []byte(fmt.Sprintf(`{"sourceRoot":%q}`, sourceRoot)))
	requireStatus(run, http.StatusOK, "run")
	if strings.Contains(run.Body.String(), "source-value-not-returned") {
		t.Fatal("run response leaked a source record value")
	}
	var result local.Result
	if err := json.Unmarshal(run.Body.Bytes(), &result); err != nil || result.RecordCount != 1 {
		t.Fatalf("run result=%+v err=%v", result, err)
	}
	replayed := request(http.MethodPost, "/api/v1/replays",
		[]byte(fmt.Sprintf(`{"sourceRoot":%q,"runId":%q,"reason":"operator_request"}`, sourceRoot, result.RunID)))
	requireStatus(replayed, http.StatusOK, "replay")
	if strings.Contains(replayed.Body.String(), "source-value-not-returned") {
		t.Fatal("replay response leaked a source record value")
	}

	cleaningRequest := []byte(`{"record":{"name":"  sample  "},"classification":"internal","correction":{"id":"trim-name","version":"v1","operation":"trim","field":"name"}}`)
	preview := request(http.MethodPost, "/api/v1/cleaning/preview", cleaningRequest)
	requireStatus(preview, http.StatusOK, "clean preview")
	var cleanResult local.CleaningResult
	if err := json.Unmarshal(preview.Body.Bytes(), &cleanResult); err != nil || !cleanResult.Preview || cleanResult.Record["name"] != "sample" {
		t.Fatalf("clean preview=%+v err=%v", cleanResult, err)
	}
	var classified map[string]any
	if err := json.Unmarshal(preview.Body.Bytes(), &classified); err != nil || classified["classification"] != "internal" {
		t.Fatalf("record-bearing response omitted its classification: %v %s", err, preview.Body.String())
	}
	applied := request(http.MethodPost, "/api/v1/cleaning/apply", cleaningRequest)
	requireStatus(applied, http.StatusOK, "clean apply")
	if err := json.Unmarshal(applied.Body.Bytes(), &cleanResult); err != nil || cleanResult.Preview || cleanResult.AlreadyApplied {
		t.Fatalf("clean apply=%+v err=%v", cleanResult, err)
	}

	largeInteger := []byte(`{"record":{"count":null},"classification":"public","correction":{"id":"replace-count","version":"v1","operation":"replace_null","field":"count","value":9007199254740993}}`)
	numericPreview := request(http.MethodPost, "/api/v1/cleaning/preview", largeInteger)
	requireStatus(numericPreview, http.StatusOK, "numeric cleaning preview")
	if !strings.Contains(numericPreview.Body.String(), `"count":9007199254740993`) {
		t.Fatalf("cleaning JSON number lost precision: %s", numericPreview.Body.String())
	}
}

func contains(values []string, match string) bool {
	for _, value := range values {
		if value == match {
			return true
		}
	}
	return false
}
