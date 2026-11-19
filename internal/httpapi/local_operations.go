package httpapi

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"time"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/local"
	"github.com/kumerian/mimir/internal/metadata"
	"github.com/kumerian/mimir/internal/ports"
)

// LocalOperations composes the existing local-profile use cases. StateDir and
// the local caller grant are fixed when the HTTP process starts.
type LocalOperations struct {
	stateDir   string
	allowLocal bool
	health     ProfileHealth
	repository ports.MetadataRepository
	reports    ports.MetadataValidationStore
	proposals  ports.MetadataProposalStore
	sourceRoot string
}

func NewLocalOperations(stateDir string, allowLocal bool, profile ProfileHealth) (*LocalOperations, error) {
	return NewLocalOperationsWithStores(stateDir, allowLocal, profile, nil, nil, nil)
}

func NewLocalOperationsWithStores(
	stateDir string,
	allowLocal bool,
	profile ProfileHealth,
	repository ports.MetadataRepository,
	reports ports.MetadataValidationStore,
	proposals ports.MetadataProposalStore,
) (*LocalOperations, error) {
	return newLocalOperations(stateDir, allowLocal, profile, repository, reports, proposals, "")
}

func NewContainerLocalOperationsWithStores(
	stateDir string,
	allowLocal bool,
	profile ProfileHealth,
	repository ports.MetadataRepository,
	reports ports.MetadataValidationStore,
	proposals ports.MetadataProposalStore,
	sourceRoot string,
) (*LocalOperations, error) {
	if !filepath.IsAbs(sourceRoot) || filepath.Clean(sourceRoot) != sourceRoot {
		return nil, domain.NewFailure(domain.FailureInvalidInput, "http.start", "an absolute clean source directory is required")
	}
	return newLocalOperations(stateDir, allowLocal, profile, repository, reports, proposals, sourceRoot)
}

func newLocalOperations(
	stateDir string,
	allowLocal bool,
	profile ProfileHealth,
	repository ports.MetadataRepository,
	reports ports.MetadataValidationStore,
	proposals ports.MetadataProposalStore,
	sourceRoot string,
) (*LocalOperations, error) {
	if !allowLocal {
		return nil, domain.NewFailure(domain.FailureAccessDenied, "http.start", "explicit local authorization is required at startup")
	}
	if stateDir == "" || !filepath.IsAbs(stateDir) {
		return nil, domain.NewFailure(domain.FailureInvalidInput, "http.start", "an absolute state directory is required")
	}
	if (repository == nil) != (reports == nil) || (repository == nil) != (proposals == nil) {
		return nil, domain.NewFailure(domain.FailureInvalidInput, "http.start", "metadata persistence adapters must be configured together")
	}
	return &LocalOperations{
		stateDir: filepath.Clean(stateDir), allowLocal: true, health: profile,
		repository: repository, reports: reports, proposals: proposals,
		sourceRoot: sourceRoot,
	}, nil
}

func (o *LocalOperations) Health(ctx context.Context) (ProfileHealth, error) {
	if err := ctx.Err(); err != nil {
		return ProfileHealth{}, domain.ContextFailure("profile.health", err)
	}
	return o.health, nil
}

func (o *LocalOperations) Validate(ctx context.Context, raw json.RawMessage) (metadata.ValidationReport, error) {
	return local.ValidateMetadataWithStore(ctx, raw, o.stateDir, o.reports)
}

func (o *LocalOperations) ValidationReport(ctx context.Context, digest string) (metadata.ValidationReport, error) {
	return local.LoadMetadataValidationReportWithStore(ctx, o.stateDir, digest, o.reports)
}

func (o *LocalOperations) Author(ctx context.Context, raw json.RawMessage) (local.RevisionSummary, error) {
	return local.AuthorBytesWithStores(ctx, raw, o.stateDir, "local-user", o.allowLocal, o.repository, o.reports)
}

func (o *LocalOperations) ListMetadata(ctx context.Context) ([]local.RevisionSummary, error) {
	if err := ctx.Err(); err != nil {
		return nil, domain.ContextFailure("metadata.list", err)
	}
	return local.ListMetadataWithRepository(ctx, o.stateDir, "local-user", o.allowLocal, o.repository)
}

func (o *LocalOperations) Activate(ctx context.Context, request ActivateRequest) (local.ActivationResult, error) {
	if request.Revision != "" {
		return local.ActivateDraftWithRepository(ctx, request.Revision, o.stateDir, "local-user", o.allowLocal, request.ExpectedActive, o.repository)
	}
	return local.ActivateBytesWithRepository(ctx, request.Metadata, o.stateDir, "local-user", o.allowLocal, o.repository, request.ExpectedActive)
}

func (o *LocalOperations) Discover(ctx context.Context, request DiscoveryRequest) (local.DiscoveryResult, error) {
	if err := o.validateSourceRoot(request.SourceRoot); err != nil {
		return local.DiscoveryResult{}, err
	}
	disposition := request.Disposition
	if disposition == "" {
		disposition = "hold"
	}
	return local.Discover(ctx, local.DiscoveryOptions{
		SourceRoot: request.SourceRoot, StateDir: o.stateDir, Actor: "local-user",
		AllowLocal: o.allowLocal, Disposition: disposition,
		MetadataRepository: o.repository, ProposalStore: o.proposals,
	})
}

func (o *LocalOperations) Run(ctx context.Context, request RunRequest) (local.Result, error) {
	if err := o.validateSourceRoot(request.SourceRoot); err != nil {
		return local.Result{}, err
	}
	return local.Run(ctx, local.Options{
		SourceRoot: request.SourceRoot, StateDir: o.stateDir, Actor: "local-user", AllowLocal: o.allowLocal,
		MetadataRepository: o.repository,
	})
}

func (o *LocalOperations) EvaluateSchedules(ctx context.Context, request ScheduleRequest) (local.ScheduleResult, error) {
	var at time.Time
	if request.At != "" {
		parsed, err := time.Parse(time.RFC3339, request.At)
		if err != nil {
			return local.ScheduleResult{}, domain.NewFailure(domain.FailureInvalidInput, "schedule.evaluate", "evaluation time must be RFC3339")
		}
		at = parsed
	}
	return local.EvaluateSchedules(ctx, local.ScheduleOptions{
		StateDir: o.stateDir, Actor: "local-user", AllowLocal: o.allowLocal, At: at,
		MetadataRepository: o.repository,
	})
}

func (o *LocalOperations) StopScheduleRequests(ctx context.Context, request StopScheduleRequest) error {
	reason := request.Reason
	if reason == "" {
		reason = "shutdown"
	}
	return local.StopPendingScheduleRequests(ctx, o.stateDir, "local-user", o.allowLocal, reason)
}

func (o *LocalOperations) Replay(ctx context.Context, request ReplayRequest) (local.ReplayResult, error) {
	if err := o.validateSourceRoot(request.SourceRoot); err != nil {
		return local.ReplayResult{}, err
	}
	return local.Replay(ctx, local.Options{
		SourceRoot: request.SourceRoot, StateDir: o.stateDir, Actor: "local-user", AllowLocal: o.allowLocal,
		MetadataRepository: o.repository,
	}, request.RunID, request.Reason)
}

func (o *LocalOperations) Clean(ctx context.Context, request CleaningRequest, preview bool) (local.CleaningResult, error) {
	return local.CleanRecordBytes(ctx, local.CleaningOptions{
		StateDir: o.stateDir, Actor: "local-user",
		Classification: request.Classification, AllowLocal: o.allowLocal, Preview: preview,
		Correction: request.Correction,
	}, request.Record)
}

func (o *LocalOperations) validateSourceRoot(requested string) error {
	if o.sourceRoot == "" {
		return nil
	}
	if !filepath.IsAbs(requested) {
		return domain.NewFailure(domain.FailureInvalidInput, "source.root", "source root must be within the mounted source directory")
	}
	relative, err := filepath.Rel(o.sourceRoot, filepath.Clean(requested))
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return domain.NewFailure(domain.FailureAccessDenied, "source.root", "source root must be within the mounted source directory")
	}
	return nil
}

var _ Operations = (*LocalOperations)(nil)
