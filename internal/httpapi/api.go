// Package httpapi implements the local, same-origin HTTP boundary.
package httpapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/kumerian/mimir/internal/cleaning"
	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/local"
	"github.com/kumerian/mimir/internal/metadata"
)

const (
	ContractVersion = "mimir.gui-api/v1"
	versionHeader   = "Mimir-Contract-Version"
	correlationHead = "X-Correlation-ID"

	metadataLimit = 1 << 20
	cleaningLimit = 256 << 10
	envelopeSlack = 4 << 10
	defaultLimit  = 64 << 10
)

var correlationPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

type ProfileHealth struct {
	Status    string   `json:"status"`
	Version   string   `json:"version"`
	Commit    string   `json:"commit"`
	Contracts []string `json:"contracts"`
	Profile   string   `json:"profile"`
}

type ActivateRequest struct {
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	Revision       string          `json:"revision,omitempty"`
	ExpectedActive string          `json:"expectedActive,omitempty"`
}

type DiscoveryRequest struct {
	SourceRoot  string `json:"sourceRoot"`
	Disposition string `json:"disposition,omitempty"`
}

type RunRequest struct {
	SourceRoot string `json:"sourceRoot"`
}

type ScheduleRequest struct {
	At string `json:"at,omitempty"`
}

type StopScheduleRequest struct {
	Reason string `json:"reason,omitempty"`
}

type ReplayRequest struct {
	SourceRoot string `json:"sourceRoot"`
	RunID      string `json:"runId"`
	Reason     string `json:"reason"`
}

type CleaningRequest struct {
	Record         json.RawMessage     `json:"record"`
	Classification string              `json:"classification"`
	Correction     cleaning.Correction `json:"correction"`
}

// CleaningResponse is intentionally record-bearing and carries the caller's
// classification for downstream display and handling.
type CleaningResponse struct {
	Record         map[string]any `json:"record"`
	Audit          cleaning.Audit `json:"audit"`
	Preview        bool           `json:"preview"`
	AlreadyApplied bool           `json:"alreadyApplied"`
	Classification string         `json:"classification"`
}

// Operations is the narrow application boundary used by the HTTP adapter.
type Operations interface {
	Health(context.Context) (ProfileHealth, error)
	Validate(context.Context, json.RawMessage) (metadata.ValidationReport, error)
	ValidationReport(context.Context, string) (metadata.ValidationReport, error)
	Author(context.Context, json.RawMessage) (local.RevisionSummary, error)
	ListMetadata(context.Context) ([]local.RevisionSummary, error)
	Activate(context.Context, ActivateRequest) (local.ActivationResult, error)
	Discover(context.Context, DiscoveryRequest) (local.DiscoveryResult, error)
	Run(context.Context, RunRequest) (local.Result, error)
	EvaluateSchedules(context.Context, ScheduleRequest) (local.ScheduleResult, error)
	StopScheduleRequests(context.Context, StopScheduleRequest) error
	Replay(context.Context, ReplayRequest) (local.ReplayResult, error)
	Clean(context.Context, CleaningRequest, bool) (local.CleaningResult, error)
}

// NewHandler creates the local-profile HTTP API handler.
func NewHandler(operations Operations) http.Handler {
	return NewHandlerWithReadiness(operations, nil)
}

// NewHandlerWithReadiness adds a separate readiness probe without changing
// the profile-identity semantics of /api/v1/health.
func NewHandlerWithReadiness(operations Operations, readiness func(context.Context) error) http.Handler {
	mux := http.NewServeMux()
	handler := apiHandler{operations: operations, readiness: readiness}
	mux.HandleFunc("GET /readyz", handler.ready)
	mux.HandleFunc("GET /api/v1/health", handler.health)
	mux.HandleFunc("POST /api/v1/metadata/validate", handler.validate)
	mux.HandleFunc("GET /api/v1/metadata/validation-reports/{inputDigest}", handler.validationReport)
	mux.HandleFunc("POST /api/v1/metadata/revisions", handler.author)
	mux.HandleFunc("GET /api/v1/metadata/revisions", handler.listMetadata)
	mux.HandleFunc("POST /api/v1/metadata/activate", handler.activate)
	mux.HandleFunc("POST /api/v1/discovery", handler.discover)
	mux.HandleFunc("POST /api/v1/runs", handler.run)
	mux.HandleFunc("POST /api/v1/schedules/evaluate", handler.evaluateSchedules)
	mux.HandleFunc("POST /api/v1/schedules/stop", handler.stopScheduleRequests)
	mux.HandleFunc("POST /api/v1/replays", handler.replay)
	mux.HandleFunc("POST /api/v1/cleaning/preview", handler.clean(true))
	mux.HandleFunc("POST /api/v1/cleaning/apply", handler.clean(false))
	mux.HandleFunc("/api/v1", handler.notFound)
	mux.HandleFunc("/api/v1/", handler.notFound)
	return handler.wrap(mux)
}

// ValidateListenAddress permits only an explicit IP-literal loopback address.
func ValidateListenAddress(address string) error {
	return ValidateListenAddressMode(address, false)
}

// ValidateListenAddressMode permits 0.0.0.0 only for the explicit container
// profile; all other non-loopback listener addresses remain rejected.
func ValidateListenAddressMode(address string, containerNetwork bool) error {
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("listen address must be an IP-literal host and port")
	}
	ip := net.ParseIP(host)
	port, portErr := strconv.Atoi(portText)
	allowedHost := ip != nil && ip.IsLoopback()
	if containerNetwork && host == "0.0.0.0" {
		allowedHost = true
	}
	if !allowedHost || portErr != nil || port < 1 || port > 65535 {
		return fmt.Errorf("listen address must use an allowed IP and a valid TCP port")
	}
	return nil
}

type apiHandler struct {
	operations Operations
	readiness  func(context.Context) error
}

func (h apiHandler) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID, err := requestCorrelationID(r)
		if err != nil {
			writeFailure(w, domain.NewFailure(domain.FailureInvalidInput, "http.request", "correlation identifier is invalid"), "")
			return
		}
		w.Header().Set(correlationHead, correlationID)
		w.Header().Set(versionHeader, ContractVersion)

		if !strings.HasPrefix(r.URL.Path, "/api/v1/") && r.URL.Path != "/api/v1" {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodOptions {
			writeFailure(w, domain.NewFailure(domain.FailureInvalidInput, "http.method", "request method is not supported"), correlationID)
			return
		}
		if !methodAllowed(r.URL.Path, r.Method) {
			writeFailure(w, domain.NewFailure(domain.FailureInvalidInput, "http.method", "request method is not supported for this operation"), correlationID)
			return
		}
		if r.Method == http.MethodPost {
			if err := sameOrigin(r); err != nil {
				writeFailure(w, domain.NewFailure(domain.FailureAccessDenied, "http.origin", "mutating requests require a same-origin loopback origin"), correlationID)
				return
			}
		}
		version := r.Header.Get(versionHeader)
		if version == "" {
			writeFailure(w, domain.NewFailure(domain.FailureInvalidInput, "http.version", "contract version header is required"), correlationID)
			return
		}
		if version != ContractVersion {
			writeFailure(w, domain.NewFailure(domain.FailureUnsupportedCapability, "http.version", "requested API contract version is not supported"), correlationID)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func methodAllowed(route, method string) bool {
	if method != http.MethodGet && method != http.MethodPost {
		return false
	}
	switch route {
	case "/api/v1/health", "/api/v1/metadata/revisions":
		return method == http.MethodGet || (route == "/api/v1/metadata/revisions" && method == http.MethodPost)
	case "/api/v1/metadata/validation-reports/":
		return method == http.MethodGet
	}
	if strings.HasPrefix(route, "/api/v1/metadata/validation-reports/") {
		return method == http.MethodGet
	}
	switch route {
	case "/api/v1/metadata/validate", "/api/v1/metadata/activate", "/api/v1/discovery",
		"/api/v1/runs", "/api/v1/schedules/evaluate", "/api/v1/schedules/stop",
		"/api/v1/replays", "/api/v1/cleaning/preview", "/api/v1/cleaning/apply":
		return method == http.MethodPost
	default:
		return false
	}
}

func requestCorrelationID(r *http.Request) (string, error) {
	if id := r.Header.Get(correlationHead); id != "" {
		if !correlationPattern.MatchString(id) {
			return "", errors.New("invalid correlation identifier")
		}
		return id, nil
	}
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}

func sameOrigin(r *http.Request) error {
	originHeader := r.Header.Get("Origin")
	if originHeader == "" {
		return errors.New("origin is missing")
	}
	origin, err := url.Parse(originHeader)
	if err != nil || origin.User != nil || origin.Path != "" || origin.RawQuery != "" ||
		origin.Fragment != "" || origin.Scheme != "http" || !strings.EqualFold(origin.Host, r.Host) {
		return errors.New("origin does not match request host")
	}
	host := origin.Hostname()
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("origin is not loopback")
	}
	return nil
}

func (h apiHandler) health(w http.ResponseWriter, r *http.Request) {
	h.invoke(w, r, "profile.health", func() (any, error) { return h.operations.Health(r.Context()) })
}

func (h apiHandler) ready(w http.ResponseWriter, r *http.Request) {
	if h.readiness != nil {
		if err := h.readiness(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, struct {
				Status string `json:"status"`
			}{Status: "not_ready"})
			return
		}
	}
	writeJSON(w, http.StatusOK, struct {
		Status string `json:"status"`
	}{Status: "ready"})
}

func (h apiHandler) notFound(w http.ResponseWriter, r *http.Request) {
	h.fail(w, r, domain.NewFailure(domain.FailureInvalidInput, "http.route", "API operation is not supported"))
}

func (h apiHandler) validate(w http.ResponseWriter, r *http.Request) {
	raw, ok := h.readBody(w, r, "metadata.validate", metadataLimit, true)
	if !ok {
		return
	}
	if err := metadata.ValidateJSONDocument(raw); err != nil {
		h.fail(w, r, domain.NewFailure(domain.FailureInvalidInput, "metadata.validate", "metadata input is not valid JSON"))
		return
	}
	h.invoke(w, r, "metadata.validate", func() (any, error) {
		return h.operations.Validate(r.Context(), json.RawMessage(raw))
	})
}

func (h apiHandler) validationReport(w http.ResponseWriter, r *http.Request) {
	h.invoke(w, r, "metadata.validation-report", func() (any, error) {
		return h.operations.ValidationReport(r.Context(), r.PathValue("inputDigest"))
	})
}

func (h apiHandler) author(w http.ResponseWriter, r *http.Request) {
	raw, ok := h.readBody(w, r, "metadata.author", metadataLimit, true)
	if !ok {
		return
	}
	if err := metadata.ValidateJSONDocument(raw); err != nil {
		h.fail(w, r, domain.NewFailure(domain.FailureInvalidInput, "metadata.author", "metadata input is not valid JSON"))
		return
	}
	h.invokeStatus(w, r, "metadata.author", http.StatusCreated, func() (any, error) {
		return h.operations.Author(r.Context(), json.RawMessage(raw))
	})
}

func (h apiHandler) listMetadata(w http.ResponseWriter, r *http.Request) {
	h.invoke(w, r, "metadata.list", func() (any, error) {
		return h.operations.ListMetadata(r.Context())
	})
}

func (h apiHandler) activate(w http.ResponseWriter, r *http.Request) {
	raw, ok := h.readBody(w, r, "metadata.activate", metadataLimit+envelopeSlack, false)
	if !ok {
		return
	}
	var request ActivateRequest
	if !decodeStrict(raw, &request) {
		h.fail(w, r, domain.NewFailure(domain.FailureInvalidInput, "metadata.activate", "activation request is invalid"))
		return
	}
	if len(request.Metadata) > metadataLimit || (len(request.Metadata) != 0 && metadata.ValidateJSONDocument(request.Metadata) != nil) {
		h.fail(w, r, domain.NewFailure(domain.FailureInvalidInput, "metadata.activate", "inline metadata is invalid or exceeds its size limit"))
		return
	}
	if (len(request.Metadata) == 0) == (request.Revision == "") {
		h.fail(w, r, domain.NewFailure(domain.FailureInvalidInput, "metadata.activate", "provide exactly one inline document or revision identifier"))
		return
	}
	h.invoke(w, r, "metadata.activate", func() (any, error) {
		return h.operations.Activate(r.Context(), request)
	})
}

func (h apiHandler) discover(w http.ResponseWriter, r *http.Request) {
	raw, ok := h.readBody(w, r, "metadata.discover", defaultLimit, false)
	if !ok {
		return
	}
	var request DiscoveryRequest
	if !decodeStrict(raw, &request) || request.SourceRoot == "" {
		h.fail(w, r, domain.NewFailure(domain.FailureInvalidInput, "metadata.discover", "discovery request requires a source root"))
		return
	}
	h.invoke(w, r, "metadata.discover", func() (any, error) {
		return h.operations.Discover(r.Context(), request)
	})
}

func (h apiHandler) run(w http.ResponseWriter, r *http.Request) {
	raw, ok := h.readBody(w, r, "run.execute", defaultLimit, false)
	if !ok {
		return
	}
	var request RunRequest
	if !decodeStrict(raw, &request) || request.SourceRoot == "" {
		h.fail(w, r, domain.NewFailure(domain.FailureInvalidInput, "run.execute", "run request requires a source root"))
		return
	}
	h.invoke(w, r, "run.execute", func() (any, error) {
		return h.operations.Run(r.Context(), request)
	})
}

func (h apiHandler) evaluateSchedules(w http.ResponseWriter, r *http.Request) {
	raw, ok := h.readBody(w, r, "schedule.evaluate", defaultLimit, false)
	if !ok {
		return
	}
	var request ScheduleRequest
	if !decodeStrict(raw, &request) {
		h.fail(w, r, domain.NewFailure(domain.FailureInvalidInput, "schedule.evaluate", "schedule evaluation request is invalid"))
		return
	}
	h.invoke(w, r, "schedule.evaluate", func() (any, error) {
		return h.operations.EvaluateSchedules(r.Context(), request)
	})
}

func (h apiHandler) stopScheduleRequests(w http.ResponseWriter, r *http.Request) {
	raw, ok := h.readBody(w, r, "schedule.stop", defaultLimit, false)
	if !ok {
		return
	}
	var request StopScheduleRequest
	if !decodeStrict(raw, &request) {
		h.fail(w, r, domain.NewFailure(domain.FailureInvalidInput, "schedule.stop", "schedule stop request is invalid"))
		return
	}
	h.invoke(w, r, "schedule.stop", func() (any, error) {
		if err := h.operations.StopScheduleRequests(r.Context(), request); err != nil {
			return nil, err
		}
		return struct {
			Status string `json:"status"`
		}{Status: "pending_requests_stopped"}, nil
	})
}

func (h apiHandler) replay(w http.ResponseWriter, r *http.Request) {
	raw, ok := h.readBody(w, r, "replay.prepare", defaultLimit, false)
	if !ok {
		return
	}
	var request ReplayRequest
	if !decodeStrict(raw, &request) || request.SourceRoot == "" || request.RunID == "" || request.Reason == "" {
		h.fail(w, r, domain.NewFailure(domain.FailureInvalidInput, "replay.prepare", "replay request requires source root, run ID, and reason"))
		return
	}
	h.invoke(w, r, "replay.prepare", func() (any, error) {
		return h.operations.Replay(r.Context(), request)
	})
}

func (h apiHandler) clean(preview bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := cleaningLimit + envelopeSlack
		raw, ok := h.readBody(w, r, "cleaning.apply", limit, false)
		if !ok {
			return
		}
		var request CleaningRequest
		if !decodeStrict(raw, &request) || len(request.Record) == 0 || len(request.Record) > cleaningLimit {
			h.fail(w, r, domain.NewFailure(domain.FailureInvalidInput, "cleaning.input", "cleaning request record is missing or exceeds its size limit"))
			return
		}
		if err := metadata.ValidateJSONDocument(request.Record); err != nil {
			h.fail(w, r, domain.NewFailure(domain.FailureInvalidInput, "cleaning.input", "cleaning record is not valid JSON"))
			return
		}
		h.invoke(w, r, "cleaning.apply", func() (any, error) {
			result, err := h.operations.Clean(r.Context(), request, preview)
			if err != nil {
				return nil, err
			}
			return CleaningResponse{
				Record: result.Record, Audit: result.Audit, Preview: result.Preview,
				AlreadyApplied: result.AlreadyApplied, Classification: request.Classification,
			}, nil
		})
	}
}

func (h apiHandler) readBody(w http.ResponseWriter, r *http.Request, operation string, limit int, document bool) ([]byte, bool) {
	if err := requireJSON(r); err != nil {
		h.fail(w, r, domain.NewFailure(domain.FailureInvalidInput, operation, "request content type must be application/json"))
		return nil, false
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, int64(limit)+1))
	if err != nil || len(raw) > limit {
		h.fail(w, r, domain.NewFailure(domain.FailureInvalidInput, operation, "request body is malformed or exceeds its byte limit"))
		return nil, false
	}
	if document {
		return raw, true
	}
	if err := metadata.ValidateJSONDocument(raw); err != nil {
		h.fail(w, r, domain.NewFailure(domain.FailureInvalidInput, operation, "request body is not valid JSON"))
		return nil, false
	}
	return raw, true
}

func requireJSON(r *http.Request) error {
	value := r.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(value)
	if err != nil || mediaType != "application/json" {
		return errors.New("not JSON")
	}
	if charset := params["charset"]; charset != "" && !strings.EqualFold(charset, "utf-8") {
		return errors.New("unsupported charset")
	}
	return nil
}

func decodeStrict(raw []byte, target any) bool {
	if err := metadata.ValidateJSONDocument(raw); err != nil {
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return false
	}
	var trailing any
	return decoder.Decode(&trailing) == io.EOF
}

func (h apiHandler) invoke(w http.ResponseWriter, r *http.Request, operation string, call func() (any, error)) {
	h.invokeStatus(w, r, operation, http.StatusOK, call)
}

func (h apiHandler) invokeStatus(w http.ResponseWriter, r *http.Request, operation string, successStatus int, call func() (any, error)) {
	if h.operations == nil {
		h.fail(w, r, domain.NewFailure(domain.FailureInternal, operation, "local operation is unavailable"))
		return
	}
	result, err := call()
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, successStatus, result)
}

func (h apiHandler) fail(w http.ResponseWriter, r *http.Request, err error) {
	id := w.Header().Get(correlationHead)
	writeFailure(w, err, id)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeFailure(w http.ResponseWriter, err error, correlationID string) {
	failure := safeFailure(err)
	status := statusFor(failure.Category)
	if correlationID != "" {
		w.Header().Set(correlationHead, correlationID)
	}
	w.Header().Set(versionHeader, ContractVersion)
	writeJSON(w, status, failure)
}

func safeFailure(err error) domain.Failure {
	var failure domain.Failure
	if !errors.As(err, &failure) {
		var failurePointer *domain.Failure
		if !errors.As(err, &failurePointer) || failurePointer == nil {
			return domain.NewFailure(domain.FailureInternal, "http.handler", "request could not be completed")
		}
		failure = *failurePointer
	}
	if failure.ContractVersion != domain.FailureContractVersion || !knownFailureCategory(failure.Category) ||
		(failure.CommitCertainty != domain.CommitNotCommitted &&
			failure.CommitCertainty != domain.CommitCommitted &&
			failure.CommitCertainty != domain.CommitUnknown) {
		return domain.NewFailure(domain.FailureInternal, "http.handler", "request could not be completed")
	}
	if failure.Category == domain.FailureAmbiguousCommit && failure.CommitCertainty != domain.CommitUnknown {
		failure.CommitCertainty = domain.CommitUnknown
		failure.Reconciliation = "reconcile_before_retry"
		failure.Retryable = false
	}
	return failure
}

func knownFailureCategory(category domain.FailureCategory) bool {
	switch category {
	case domain.FailureInvalidInput, domain.FailureUnsupportedCapability, domain.FailureAccessDenied,
		domain.FailureConflict, domain.FailurePolicyRejected, domain.FailureUnavailable,
		domain.FailureAmbiguousCommit, domain.FailureCancelled, domain.FailureTimeout, domain.FailureInternal:
		return true
	default:
		return false
	}
}

func statusFor(category domain.FailureCategory) int {
	switch category {
	case domain.FailureInvalidInput:
		return http.StatusBadRequest
	case domain.FailureUnsupportedCapability, domain.FailurePolicyRejected:
		return http.StatusUnprocessableEntity
	case domain.FailureAccessDenied:
		return http.StatusForbidden
	case domain.FailureConflict:
		return http.StatusConflict
	case domain.FailureUnavailable:
		return http.StatusServiceUnavailable
	case domain.FailureAmbiguousCommit, domain.FailureInternal:
		return http.StatusInternalServerError
	case domain.FailureCancelled:
		return http.StatusRequestTimeout
	case domain.FailureTimeout:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}
