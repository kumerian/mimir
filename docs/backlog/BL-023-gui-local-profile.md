# BL-023 refinement — Angular GUI for the local reference profile

Parent: [BL-023](./backlog.md#bl-023-deliver-an-angular-gui-for-the-implemented-local-reference-profile)
Status: In Progress
Architecture: [GUI architecture](../architecture/gui-architecture.md), [software architecture](../architecture/software-architecture.md), [system architecture](../architecture/system-architecture.md)
Related entrypoint: [BL-019](./BL-019-entrypoints.md)
Workflow: [change implementation standard](../implementation/change-workflow.md)

## Outcome and current implementation evidence

**Required outcome:** provide an Angular presentation that communicates with
Go backend endpoints and covers only the currently implemented local
reference-profile operations. The GUI is a presentation adapter; existing
Go use cases retain validation, authorization, policy, planning, run-state,
audit, data, and recovery authority.

**Current:** `cmd/mimir/main.go` implements the CLI commands and a gated
`serve` entrypoint; `internal/httpapi/` exposes the accepted local API through
existing `internal/local/` use cases. The OpenAPI 3.1 contract is
`contracts/openapi/local-profile-v1.json`. A minimal Angular profile and API
connectivity shell, Nginx proxy, three-service Compose profile, and MariaDB
metadata adapter are implemented. Full operation screens, automated
backup/restore, and deployment/promotion support remain outstanding. The
process, endpoint, caller grant, and data-input boundary are accepted in
[ADR-0005](../architecture/decisions/ADR-0005-local-gui-and-http-endpoint-boundary.md).
[ADR-0006](../architecture/decisions/ADR-0006-local-compose-and-metadata-persistence.md)
accepts the implemented local Compose/MariaDB target. ADR-0004
continues to defer generic remote APIs and hosted deployment.

**Current profile surface to preserve:**

- `health` profile identity only; not filesystem or environment readiness.
- `validate` and digest-based `validation-report`; reports remain
  non-run-eligible.
- `author`, `list-metadata`, and explicit `activate`, including
  `expected-active` when replacing the active revision.
- `discover` immediate proposal result; no proposal-history query and no
  active metadata mutation.
- `run` for one bounded local JSONL snapshot; show only returned run/revision
  identity, record count, and reconciliation indicator.
- `schedule` request evaluation and `schedule-stop` request acknowledgements;
  neither dispatches work nor changes run state.
- `replay` only for reconciliation of the original pinned idempotent batch.
- `clean` preview and explicit separate application for the finite documented
  operations and classifications.

The GUI must not present run history, individual run/checkpoint state,
per-record quality results, audit/telemetry queries, destination or quarantine
browsing, cleaned-output history, or persisted proposal retrieval. Refer to
the GUI architecture capability map for the full operation limits, failure
semantics, and security constraints.

## Contracts and boundaries affected

- **Required:** Angular is a presentation adapter and calls HTTP endpoints on
  the Go backend. Go endpoint translation calls existing application use
  cases; it must not copy their domain rules.
- **Required:** endpoint request/result/error contracts are explicit and
  versioned. Existing CLI JSON output is not an endpoint contract.
- **Required:** local authorization, `--allow-local`-equivalent grant,
  classification, capability, locking, audit, and recovery checks remain
  enforced outside the UI.
- **Accepted:** same-origin Angular SPA served by Nginx with `/api/` proxied
  to Go in the container profile; synchronous HTTP/JSON under `/api/v1` with
  OpenAPI 3.1; standalone loopback-only and explicit container-network modes
  requiring `--allow-local`; fixed startup state directory and explicit source
  root for operations that need it; bounded inline JSON metadata/cleaning
  inputs; same-origin checks, bounded request bodies, request-context
  cancellation, no CORS/remote origins, and `mimir.failure/v1` with explicit
  HTTP status mappings (ADR-0005).
- **Delivery detail:** operation DTOs/body limits are specified in ADR-0005;
  frontend toolchain versions remain open.
- **Compatibility:** the file profile and its CLI/tests remain unchanged; no
  automatic migration to MariaDB is authorized. API and frontend compatibility
  need explicit versioning and rejection behavior.

## Child work items

### BL-023.1: Decide the GUI runtime, local caller, endpoint, and installation boundary

Type: Architecture decision
Status: Done
Priority: P1

Problem:
The GUI architecture left the local backend and caller boundary open.
ADR-0005 records the accepted conservative decisions and their limits. No
runtime code, installation flow, or frontend toolchain is introduced by this
decision. Its original serving/listener choices are amended by ADR-0006;
the completed criteria below record the decision as it stood at acceptance.

Scope:
- record Angular browser SPA and same-origin Go asset/API serving;
- preserve ADR-0001's supported Go build targets; make no new GUI runtime
  support claim beyond Linux/amd64, the only runtime-verified target;
- record HTTP/JSON, `/api/v1`, OpenAPI 3.1, and existing stable failure mapping;
- record single-user loopback-only backend, startup `--allow-local`, and
  existing `local-user` / allow-local use-case grant;
- record fixed state directory, explicit source root for operations requiring
  it, and bounded inline JSON for metadata/cleaning inputs;
- record no CORS/remote origins, same-origin checks for browser mutations,
  bounded JSON bodies, and request-context cancellation/deadlines;
- keep new queries, hosted identity, shared-host/multi-user service, and
  production deployment out of scope.

Out of scope:
- hosted identity, a production access-control system, new domain/use-case
  queries, or a specific deployment product absent a concrete requirement.

Affected modules and contracts:
- `docs/architecture/gui-architecture.md`, `docs/architecture/software-architecture.md`,
  `docs/architecture/decisions/ADR-0004-delivery-boundary-and-external-adapters.md`,
  new ADR-0005, and future endpoint contract.

Acceptance criteria:
- [x] Given each GUI/backend boundary question, when the decision is reviewed,
  then it has an explicit answer, owner, consequences, alternatives, status,
  and affected scope in an accepted ADR; remaining operation schema and
  toolchain details are assigned to dependent implementation work.
- [x] Given a local caller, when the selected boundary is documented, then it
  states how the explicit local grant is conveyed and enforced without treating
  UI visibility or local profile status as authorization/readiness.
- [x] Given the accepted local boundary, when documented, then process binding,
  startup grant, state/source path ownership, request cancellation, and
  unsupported shared-host/remote exposure are unambiguous without selecting an
  infrastructure product.
- [x] Given any requested history/query outside the implemented map, when
  considered, then it is either explicitly deferred or tracked as a separate
  use-case/API decision rather than implied by BL-023.

TDD first:
- First failing test: no runtime behavior changes in this decision-only item.
  First validation is a decision-coverage checklist against the accepted
  boundary plus `make docs` for ADR structure and links; executable
  contract tests begin in BL-023.2 after the decision.
- Additional unit/contract/integration/acceptance coverage: not applicable
  until the decision creates an executable contract.

Implementation notes:
- The technology baseline's OpenAPI 3.1 and HTTP/JSON convention is affirmed.
  Exact operation schemas and limits are BL-023.2; this decision establishes
  the endpoint version prefix and failure envelope/mappings.
- ADR-0004 now distinguishes this local GUI boundary from deferred remote API
  and worker contracts.

Documentation updates:
- Functional: distinguish the proposed GUI from current CLI behavior.
- Installation: record the chosen form factor and supported targets; do not
  write installation steps for an undecided model.
- Operations/release: document lifecycle, endpoint exposure, shutdown,
  update/rollback, and deployment assumptions selected by the ADR.
- Architecture/ADR: resolve the open questions and update GUI/software/
  technology documentation consistently.

Security and data impact:
- Decide local caller/grant, endpoint exposure, path handling, and cleaning
  result display constraints. Keep record content, secret references, raw
  errors, and local journals outside ordinary logs.

Compatibility, migration, and rollback:
- No persisted-data change is authorized by this decision. Record API
  compatibility and UI/backend version coupling; rollback must not require
  editing local journals or metadata.

Environment verification:
- Local: review decision against CLI behavior and local profile.
- Test: verify the decision is implementable with hermetic and synthetic
  acceptance tests.
- Acceptance: identify any operator-provisioned resources needed by the chosen
  model; do not presume they exist.
- Production promotion or smoke: not applicable before an approved target,
  access policy, exact artifact, and rollback plan exist.

Dependencies:
- BL-001.
- BL-003.
- BL-005.
- BL-021.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus the
  decision-specific evidence above. ADR-0005 and synchronized architecture
  documentation satisfy this decision-only item; no runtime behavior is
  claimed.

### BL-023.2: Specify and freeze the versioned local-profile endpoint contract

Type: Technical enabler
Status: Review
Priority: P1

Problem:
The Angular client needs a stable contract for exactly the operations already
implemented by Mimir. CLI output and error text do not provide that contract.

Scope:
- define request, response, version, correlation, stable failure category,
  retryability, and commit-certainty representations for the mapped local
  operations;
- describe only profile status, validation/report lookup, author/list/activate,
  discovery, bounded run, schedule evaluation/stop, constrained replay, and
  explicit cleaning preview/apply;
- specify input limits, classification/redaction, authorization context,
  cancellation, timeouts, and which outputs may contain record content;
- add contract compatibility fixtures and API documentation from the accepted
  BL-023.1 decisions.

Out of scope:
- endpoint handlers, Angular screens, new queries, arbitrary run-history or
  record browsing, and alternate infrastructure adapters.

Affected modules and contracts:
- `contracts/` API description location selected by BL-023.1;
  `internal/domain/failure.go`; existing use cases under `internal/local/`;
  future Go API boundary and Angular client types.

Acceptance criteria:
- [x] Given each mapped GUI operation, when its contract is validated, then
  its request and response identify one existing use case and its documented
  limits; no unimplemented query is described as available.
- [x] Given invalid, denied, unsupported-version, timeout/cancellation, busy,
  corrupt-state, and ambiguous-commit outcomes, when serialized, then clients
  can distinguish stable safe categories without receiving secret values,
  raw adapter errors, record payloads, or journal contents.
- [x] Given a contract version or required field the server does not support,
  when a request is submitted, then the contract requires explicit rejection
  before side effects.
- [x] Given cleaning preview/application or run output, when serialized, then
  only the explicitly documented record-bearing cleaning response may contain
  record data and it carries its classification/handling requirements.

TDD first:
- First failing test: the initial handler test returned 404 for the expected
  health response before the route was implemented. Subsequent contract and
  guard tests cover the supported operation set, unknown versions/fields,
  unsupported queries, body limits, and stable errors.
- Additional unit/contract/integration/acceptance coverage: request/result
  serialization, compatibility fixtures, redaction allowlist, limits, and
  cancellation/error mapping.

Implementation notes:
- The contract is JSON-form OpenAPI 3.1.0 at
  `contracts/openapi/local-profile-v1.json`. The repository-pinned
  `scripts/check_openapi.py` validates its version, paths, local references,
  and operation definitions; no generated clients or schemas were needed.
- Typed endpoint DTOs map to existing local use cases. The failure envelope
  remains `mimir.failure/v1`; there is no persisted/API data migration.

Documentation updates:
- Functional: add an endpoint/user-journey reference with synthetic examples
  and unavailable-query limitations.
- Installation: document only configuration implied by the accepted contract.
- Operations/release: document versions, health limitations, redacted errors,
  timeouts, and compatibility policy.
- Architecture/ADR: link the contract and record any material boundary change.

Security and data impact:
- Threat model untrusted request fields and paths; test denied access,
  classification ceilings, redaction, bounded inputs, and explicit local
  grants. Never place secrets in requests, responses, plans, or logs.

Compatibility, migration, and rollback:
- No local metadata/journal migration. Define endpoint version rejection and
  client/backend compatibility before implementation; old CLI behavior remains
  unchanged. Roll back by restoring the prior GUI/backend artifact pair, not
  by changing state files.

Environment verification:
- Local: contract validation and deterministic fixtures.
- Test: Go contract tests against synthetic data and fake use-case ports.
- Acceptance: verify the same contract against the exact integrated candidate.
- Production promotion or smoke: only after approved environment and caller
  policy; no production profile is implied.

Dependencies:
- BL-023.1.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus the
  contract-version, compatibility, security, and documentation evidence above.

### BL-023.3: Expose existing use cases through the Go backend endpoint boundary

Type: Feature
Status: Review
Priority: P1

Problem:
The local use cases have only a CLI entrypoint. A Go endpoint boundary is
needed for Angular, but must remain thin and preserve existing use-case
semantics.

Scope:
- compose the selected backend lifecycle and HTTP handlers around existing
  application/local use cases;
- map versioned requests/results and stable failures to/from typed DTOs;
- pass explicit caller grant, context, timeout, cancellation, correlation, and
  source/state path inputs according to BL-023.1/2;
- expose only operations in the approved local-profile contract;
- surface local lock/busy, corrupt state, and ambiguous commit without
  automatic state repair or unsafe retries.

Out of scope:
- moving policy, planning, run-state transitions, storage, or retries into
  handlers; new local-profile queries or state mutation; production hosting.

Affected modules and contracts:
- `cmd/mimir/main.go`, `internal/local/`, `internal/ports/`,
  `internal/domain/failure.go`, and new endpoint/composition packages selected
  by the ADR.

Acceptance criteria:
- [x] Given any accepted endpoint request, when handled, then it invokes the
  corresponding existing use case and returns its typed result without
  implementing business transitions in the handler.
- [x] Given a missing/denied local grant, invalid metadata, unsupported
  capability, or classification failure, when the endpoint is called, then
  the request is denied before protected side effects and returns a stable
  redacted error.
- [x] Given a client disconnect, timeout, cancellation, held local lock,
  corrupt journal, or ambiguous commit, when the operation returns, then its
  existing use-case recovery semantics and uncertainty are preserved and no
  blind retry or UI-side repair occurs.
- [x] Given concurrent requests for one local state directory, when the
  existing single-process/locking contract cannot safely serve them, then the
  endpoint returns the documented busy outcome instead of bypassing ownership.
- [x] Given supported endpoint requests, when contract and security tests run,
  then response data contains no source records except the permitted explicit
  cleaning result and contains no secret material or raw adapter errors.

TDD first:
- First failing test: the initial health handler test returned 404 rather
  than 200 before the endpoint implementation. Focused tests then exercise
  local use-case flow, version/origin/input guards, cancellation, error
  mapping, redaction, and the complete supported operation set.
- Additional unit/contract/integration/acceptance coverage: generated/API
  contract validation, timeout/cancellation, lock contention, malformed
  requests, audit/redaction assertions, and synthetic end-to-end Go endpoint
  tests for every operation.

Implementation notes:
- Keep endpoint composition at the entrypoint boundary. Avoid side effects
  before contract version, request size, and authorization checks.
- Do not infer request retries from HTTP method or convert ambiguous commits
  into ordinary server errors that imply safety.
- The `serve` command requires `--allow-local`, an absolute fixed
  `--state-dir`, and an IP-literal loopback listener. Inline metadata and
  cleaning documents are staged only in server-created restrictive temporary
  files to bridge existing path-based use cases; clients cannot supply those
  paths.

Documentation updates:
- Functional: document operation/result/error behavior and CLI parity limits.
- Installation: document launch, binding, local-user permissions, and shutdown
  from BL-023.1 decisions.
- Operations/release: document readiness limitations, lock contention,
  cancellation, recovery, and redaction.
- Architecture/ADR: update entrypoint/composition responsibility if the
  implementation materially changes it.

Security and data impact:
- Exercise authorization denial, request/path validation, least exposure,
  bounded request sizes, output redaction, and non-leakage in logs/audit.
  State/source directories remain operator-owned; handlers never edit journals.

Compatibility, migration, and rollback:
- No persisted format change. Reject incompatible API versions explicitly.
  The prior CLI remains usable. Roll back the endpoint/UI artifact pair and
  verify the same local state contract before reuse.

Environment verification:
- Local: focused/race tests, full Go suite, vet, OpenAPI/docs checks, and
  loopback health/graceful-shutdown smoke have passed.
- Test: hermetic API contract and adversarial error/redaction tests.
- Acceptance: still required against an exact candidate artifact in a
  provisioned acceptance environment.
- Production promotion or smoke: blocked until environment, endpoint exposure,
  approval, and rollback policy are configured.

Dependencies:
- BL-023.2.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus endpoint
  contract, authorization, recovery, and evidence requirements above.

### BL-023.4: Deliver Angular metadata and discovery journeys

Type: Feature
Status: Blocked
Priority: P1

Problem:
Operators need a presentation for local profile identity and the implemented
metadata control-plane operations without treating UI validation as authority
or discovery as activation.

Scope:
- create the Angular application shell and typed client for the accepted
  contract;
- present profile status, validation and digest-based report lookup, valid
  draft authoring, metadata identity listing, conditional explicit activation,
  and immediate discovery proposal results;
- show safe diagnostics, digests, revision/status identity, unsupported
  metadata/rule limits, and the fact that health does not test local readiness;
- keep draft contents, diagnostics, and paths within the handling rules decided
  in BL-023.1.

Out of scope:
- client-side duplicate metadata validation/policy, editing active revisions
  in place, proposal history, metadata browsing beyond returned fields,
  automatic activation, or new API operations.

Affected modules and contracts:
- Angular application and typed API client; BL-023.2 contract; Go endpoints
  from BL-023.3; metadata v1 use cases and validation report store.

Acceptance criteria:
- [ ] Given a metadata v1 input, when validation fails, then the GUI displays
  safe returned diagnostics and `runEligible: false` without authoring or
  activating it.
- [ ] Given a valid draft, when it is authored and listed, then the GUI shows
  only returned immutable revision/status/digest identities and does not
  invent revision contents or history.
- [ ] Given a draft selected for activation, when it would replace an active
  revision, then the GUI requires the expected-current-revision value and
  displays the returned outcome; a stale replacement remains a conflict.
- [ ] Given discovery output, when shown, then it is labeled an advisory
  proposal that has not changed active metadata and no proposal-history query
  is implied.
- [ ] Given `health` reports the built-in profile registration, when shown,
  then the interface explicitly says it is not filesystem/source/destination
  readiness.

TDD first:
- First failing test: Angular component/client tests for invalid validation
  not enabling author/activate, conditional activation, stale conflict display,
  and discovery as non-mutating proposal.
- Additional unit/contract/integration/acceptance coverage: supported/unknown
  metadata versions, safe diagnostics, transport failure states, accessibility,
  and end-to-end synthetic draft/validation/activation/discovery journeys.

Implementation notes:
- The backend remains authoritative for validation, access policy, strict
  metadata v1, and activation compare-and-set behavior. GUI gating is usability
  only.

Documentation updates:
- Functional: metadata and discovery journey, prerequisites, valid/invalid
  outcomes, and non-mutation behavior.
- Installation: link the UI launch and local profile setup once selected.
- Operations/release: document safe diagnostic display and revision conflict
  recovery.
- Architecture/ADR: update only if the client boundary or operation map changes.

Security and data impact:
- Do not log full metadata, secret references, source data, or raw errors.
  Preserve local classification policy; no hosted identity is claimed.

Compatibility, migration, and rollback:
- Metadata v1 and stored reports are unchanged. Display/report lookup remains
  keyed by exact input digest; GUI rollback must not rewrite metadata state.

Environment verification:
- Local: component tests and synthetic metadata input.
- Test: endpoint-backed journey with temporary private state.
- Acceptance: verify against the exact candidate and documented supported OS.
- Production promotion or smoke: not applicable to the local reference profile
  unless separately approved and provisioned.

Dependencies:
- BL-023.1.
- BL-023.2.
- BL-023.3.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus all journey,
  accessibility, and data-handling evidence above.

### BL-023.5: Deliver Angular run, schedule-request, and replay journeys

Type: Feature
Status: Blocked
Priority: P1

Problem:
Operators need explicit controls for the bounded local run and implemented
schedule/replay requests, with precise distinction between request handling
and data execution.

Scope:
- provide bounded run input and display only returned run ID, pinned metadata
  revision, count, and already-committed/reconciliation indicator;
- evaluate fixed-interval schedules and show returned decisions/request IDs;
- expose the explicit action that stops pending local schedule requests and
  label it as distinct from cancelling or stopping a run;
- request constrained replay for an existing run with required reason and
  display that it reconciles the original batch;
- surface cancellation, timeout, unsupported capability, busy, recovery, and
  ambiguous-commit outcomes without implicit retries.

Out of scope:
- scheduling/dispatching runs, run history, run detail/checkpoint queries,
  source-record browsing, arbitrary/historical replay, or creating new replay
  output.

Affected modules and contracts:
- Angular client and BL-023.2 contract; Go endpoints/ existing
  `internal/local/run.go`, `scheduler.go`, `replay.go`; source/destination,
  run-state, audit, and telemetry contracts.

Acceptance criteria:
- [ ] Given a configured local source and active revision, when a bounded run
  completes, then the GUI shows only the documented summary and never infers
  success from `health` or a schedule request.
- [ ] Given schedule evaluation, when requests are returned, then the GUI
  presents them as undelivered requests and never offers a dispatch action.
- [ ] Given a stop action, when pending requests are acknowledged, then the GUI
  describes request stopping and does not describe a run as cancelled or
  change run state.
- [ ] Given replay is requested, when original metadata/source/capability/rule
  checks fail or reconciliation is ambiguous, then the GUI presents the
  returned explicit failure without falling back to current metadata, editing
  source/state, or blindly retrying.
- [ ] Given an operation times out or is cancelled, when its outcome is
  uncertain, then the GUI reports the contract's certainty/recovery guidance
  and does not turn uncertainty into success.

TDD first:
- First failing test: component tests proving schedule responses cannot be
  dispatched, stop requests cannot be labeled run cancellation, and replay
  failure/ambiguity cannot trigger automatic retry.
- Additional unit/contract/integration/acceptance coverage: bounded run
  summary redaction, timeout/cancellation, lock contention, safe replay
  preconditions, duplicate/reconciliation behavior, and synthetic end-to-end
  journeys.

Implementation notes:
- Run/source bounds, scheduling semantics, replay guarantees, and recovery
  remain those of the local use cases. Keep output free of source values.

Documentation updates:
- Functional: run limits, request-versus-run distinctions, replay guarantees,
  cancellation, ambiguous outcomes, and recovery guidance.
- Installation: source/state path permissions and required local grant.
- Operations/release: lock ownership, unchanged-input reconciliation, backup,
  and documented escalation for corrupt/ambiguous state.
- Architecture/ADR: document any contract change; do not claim new scheduling
  or replay guarantees.

Security and data impact:
- Treat all paths and request fields as untrusted. Preserve local grant,
  classification, safe error, audit, and no-record-output requirements.

Compatibility, migration, and rollback:
- No run-state/checkpoint/data migration. Do not retry across a version change
  unless the endpoint contract and exact use case guarantee it is safe.
  Rollback preserves the complete state directory.

Environment verification:
- Local: synthetic JSONL source and private temporary state.
- Test: cancellation, duplicate request, lock, and ambiguous-commit fixtures.
- Acceptance: exercise all journey outcomes against the candidate artifact.
- Production promotion or smoke: no production use is claimed; any future
  deployment requires explicit approval and recovery rehearsal.

Dependencies:
- BL-023.1.
- BL-023.2.
- BL-023.3.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus execution
  safety, failure, and recovery evidence above.

### BL-023.6: Deliver explicit Angular cleaning preview and apply

Type: Feature
Status: Blocked
Priority: P1

Problem:
The local profile implements a narrowly scoped explicit cleaning operation.
The GUI must distinguish its sensitive preview/apply flow from validation,
quality evaluation, and ingestion, and must never mutate the original record.

Scope:
- accept one record input, explicit `public`/`internal` classification,
  correction ID/version, field, and a documented supported operation;
- display preview without persistence and require a separate deliberate apply
  action;
- show the returned cleaned result as separate output with its classification
  and audit identity/summary where the endpoint contract allows;
- state the finite operations: `trim`, `lowercase`, `uppercase`, and
  `replace_null`.

Out of scope:
- automatic cleaning, quality-triggered correction, replacing original or
  ingested data, additional classification levels or operations, and cleaned
  output/history browsing.

Affected modules and contracts:
- Angular client and BL-023.2 contract; Go endpoint and
  `internal/local/cleaning.go`; cleaning audit, classification, and local
  output contracts.

Acceptance criteria:
- [ ] Given an unsupported classification, operation, correction version, or
  unauthorized request, when submitted, then it fails closed without writing
  cleaned output.
- [ ] Given a valid preview request, when preview completes, then no durable
  cleaned record is created and the original is unchanged.
- [ ] Given a valid preview, when the operator explicitly applies the same
  correction, then output is separate, the original is unchanged, and the
  documented audit event is durable/idempotently reconciled.
- [ ] Given a cleaning result, when displayed or cleared, then the interface
  warns that record content is sensitive local data and does not send it to
  ordinary logs, telemetry, audit text, or crash reporting.
- [ ] Given the same correction request is retried after interruption, when
  the backend reconciles it, then the GUI displays the authoritative outcome
  and does not create or imply duplicate output.

TDD first:
- First failing test: component test that preview cannot invoke apply, apply
  requires explicit confirmation, and a result never overwrites the original.
- Additional unit/contract/integration/acceptance coverage: operation/version
  rejection, classification, redaction, interruption/reconciliation,
  sensitive-display cleanup, keyboard/accessibility, and endpoint-backed
  synthetic preview/apply.

Implementation notes:
- Do not persist record content in browser storage, URL state, client logs, or
  general telemetry. Use only behavior and operations already accepted by the
  local cleaning contract.

Documentation updates:
- Functional: explicit preview/apply example with synthetic record and
  operation limits.
- Installation: warn about private local state and handling of record-bearing
  responses.
- Operations/release: document separate output, audit, interrupted-apply
  reconciliation, and clearing/retention responsibilities.
- Architecture/ADR: update only if the existing cleaning boundary changes.

Security and data impact:
- This is the GUI surface with intentionally returned record content.
  Minimize display lifetime and copies, honor classification, and verify
  redaction in logs, telemetry, errors, and audit.

Compatibility, migration, and rollback:
- No source/destination/cleaned journal schema change. Rollback preserves
  original and cleaned evidence; never erase or rewrite state to undo apply.

Environment verification:
- Local: synthetic public/internal records and private temporary state.
- Test: preview/apply, denial, interruption, audit, and output-isolation tests.
- Acceptance: security-reviewed synthetic journey on the exact candidate.
- Production promotion or smoke: not applicable absent explicit production
  approval; the local profile is not production-ready.

Dependencies:
- BL-023.1.
- BL-023.2.
- BL-023.3.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus explicit
  cleaning, classification, audit, and sensitive-data evidence above.

### BL-023.7: Integrate, install, verify, and document the GUI candidate

Type: Technical enabler
Status: Blocked
Priority: P1

Problem:
An Angular UI and Go endpoint can each pass isolated checks while failing as a
local product. The complete candidate needs reproducible integration,
installation, functional guidance, artifact identity, and an honest rollback
and environment-verification record.

Scope:
- compose the approved Angular and Go backend form factor into a reproducible
  candidate without rebuilding after candidate tests/acceptance;
- add hermetic frontend and Go checks to CI, contract generation/freshness
  checks if selected, and synthetic end-to-end tests for every mapped
  capability;
- validate clean install, launch, health limitations, shutdown, local
  permissions, and rollback on each selected supported target;
- update GUI/API, functional local-profile, installation, operations, README,
  and release documentation;
- record exact source/artifact/contract identity and acceptance evidence.

Out of scope:
- production deployment, environment provisioning, signing or registry
  products not already decided, automatic file-to-database migration, or a claim that the
  local-reference profile is production suitable.

Affected modules and contracts:
- Angular and Go build configuration, CI workflow, package/artifact manifest,
  docs and synthetic acceptance tests; BL-023.2–BL-023.6 outputs; existing
  `scripts/check_tasks.py`, `scripts/check_docs.py`, `Makefile`, and
  `.github/workflows/ci.yml` as applicable.

Acceptance criteria:
- [ ] Given a clean checkout, when the documented setup/build/test tasks run,
  then pinned/selected frontend and Go tools produce a candidate with
  identifiable source revision and contract identity without ambient
  credentials.
- [ ] Given the GUI candidate, when CI executes contract, backend, frontend,
  security/redaction, accessibility, end-to-end, docs, and packaging checks,
  then all applicable results are recorded as passed, failed, not applicable,
  or explicitly externally blocked.
- [ ] Given one candidate artifact, when local/test/acceptance checks run,
  then the exact tested artifact identity is retained and production does not
  use a rebuilt or unverified artifact.
- [ ] Given install, upgrade, or rollback, when followed on selected targets,
  then endpoint exposure, local data ownership, shutdown, existing state
  compatibility, backup/recovery, and rollback behavior match the accepted
  deployment decision and documentation.
- [ ] Given documentation and generated API/schema artifacts, when repository
  doc/contract checks run, then links and generated artifacts are current and
  no documentation claims unimplemented functionality.
- [ ] Given the repository has no configured GUI acceptance/production target,
  when promotion evidence is reported, then the missing environment/approval
  is explicitly blocked and no production smoke or promotion is claimed.

TDD first:
- First failing test: add an end-to-end candidate test that starts the decided
  composition and proves health limitations, one full metadata-to-run journey,
  schedule request semantics, replay constraints, cleaning isolation, and
  absence of unsupported queries.
- Additional unit/contract/integration/acceptance coverage: clean-install and
  rollback smoke, all supported native targets, generated-contract freshness,
  artifact identity/provenance, accessibility, security, and candidate
  acceptance against the exact artifact.

Implementation notes:
- Reuse repository task conventions; do not add a package manager, CI product,
  signing service, hosting platform, or deployment target without the decisions
  in BL-023.1 and existing delivery ADRs.
- Local and CI verification do not substitute for acceptance or production
  promotion. Preserve the environment status as blocked where resources are
  absent.

Documentation updates:
- Functional: complete operation reference and synthetic journeys, supported
  inputs/limits, failures, retries, recovery, audit, and unavailable features.
- Installation: clean setup, supported OS/runtime/tool versions, permissions,
  artifact verification, launch/health, upgrade, backup, rollback, and
  uninstall for the chosen form factor.
- Operations/release: binding and lifecycle, configuration, state ownership,
  observability/redaction, deployment order, exact candidate identity,
  acceptance/smoke, rollback, limitations, and release note.
- Architecture/ADR: synchronize accepted decisions and actual composition.

Security and data impact:
- Check frontend dependency/supply-chain posture, Go/Angular compatibility,
  local endpoint exposure, request limits, authorization, secret exclusion,
  classification, record-bearing cleaning UI, and crash/log redaction.

Compatibility, migration, and rollback:
- No automatic local persistence migration is authorized. Test previous supported
  endpoint/UI contract compatibility as decided; reject unsupported versions
  before effects. Roll back to the exact prior artifact and preserve the
  complete file-profile state directory or both container-profile volumes.

Environment verification:
- Local: clean, synthetic setup/install and all documented user journeys.
- Test: isolated CI integration and clean-install artifact checks on selected
  native targets.
- Acceptance: operator verifies the exact candidate in the configured
  acceptance environment, including security, recovery, and rollback.
- Production promotion or smoke: remains blocked until target, approval,
  signing/provenance, data policy, and rollback prerequisites are established;
  no deployment is authorized by this item alone.

Dependencies:
- BL-002.
- BL-020.
- BL-021.
- BL-023.2.
- BL-023.3.
- BL-023.4.
- BL-023.5.
- BL-023.6.
- BL-023.10.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus exact-artifact,
  installation, CI/CD, acceptance, and rollback evidence above. BL-023.7
  remains blocked for any environment evidence not yet provisioned.

### BL-023.8: Implement the MariaDB metadata adapters

Type: Technical enabler
Status: In Progress
Priority: P1

Problem and scope:
- Implement the ADR-0006 `metadata` database as the container profile's sole
  authority for drafts, immutable revisions, active pointer/history,
  validation reports and discovery proposals. Adapt `MetadataRepository` and
  `MetadataValidationStore`; define a proposal persistence port and route
  discovery through it. Keep domain contracts independent of SQL.
- Preserve the file profile for CLI and existing tests. No automatic import,
  schema migration of existing file data, or new proposal query is implied.
- Use least-privilege credentials and no secret values in metadata or logs.
  Separate metadata from the backend volume's state, audit and data.

Affected modules and contracts:
- `internal/ports/` — version the metadata proposal persistence port and add
  immutable draft retrieval where activation needs it.
- `internal/local/` — inject repository, validation-report, and proposal-store
  ports while preserving the existing file-backed CLI composition.
- `internal/adapters/mariadb/` — implement MariaDB adapters and embedded,
  forward-only schema migrations for the `metadata` database.
- `internal/httpapi/` and `cmd/mimir/` — compose SQL-backed operations only in
  container mode; never fall back to file metadata when DB configuration is
  missing or unavailable.

Acceptance criteria:
- [x] Given a valid draft, when persisted and
  the database restarts, then its identity and content remain retrievable
  through the API and SQL adapter.
- [ ] Given invalid or conflicting metadata, when persisted, then the adapter
  fails closed without replacing immutable content.
- [ ] Given concurrent activation, when expected-active preconditions race,
  then at most one wins and pointer/history update atomically; validation,
  immutable revisions, provenance and pinned replay remain intact.
- [ ] Given a disconnect or uncertain commit, when an operation fails, then
  it reports a redacted stable failure and never blindly repeats a write;
  required audit outside the database remains recoverable.
- [x] Given the file profile, when existing CLI and tests run, then its
  behavior and storage remain unchanged.
- [x] Given database credentials, when a request executes, then the runtime
  account has only metadata-schema DML permissions; schema changes use a
  separate migration account and parameterized SQL.
- [ ] Given a validation report or discovery proposal, when stored, then
  only its contract payload and identity are retained; record values,
  resolved secrets, and raw adapter errors are not added to SQL metadata.

TDD first:
- First failing tests: MariaDB-backed shared port tests for draft immutability,
  revision listing and loading, conditional activation under concurrent
  expected-active races, immutable report identity, and proposal persistence.
- Additional tests: migration versioning and insufficient runtime privileges;
  transaction interruption/ambiguous commit mapping; restart persistence;
  redaction; database disconnect; and regression tests for the file profile.

Compatibility, migration, and rollback:
- Existing local files are not imported, renamed, deleted, or migrated.
  Container mode starts with a new MariaDB metadata authority and an independent
  backend execution-state volume.
- Embedded numbered migrations are forward-only and recorded in a schema
  version table. A backend image must reject an unsupported newer schema
  without mutating metadata.
- Roll back application code only to a version compatible with the recorded
  schema. Restore the offline MariaDB and backend-volume backups together if
  schema rollback is unavailable; never infer or recreate metadata from
  execution journals.

Security and data impact:
- Runtime SQL uses prepared statements and a schema-scoped DML account.
  Migration credentials are used only during startup migrations and are not
  used by operation handlers. Do not log DSNs, passwords, payloads, or query
  arguments.
- Metadata remains subject to the existing public/internal policy. Secret
  values are forbidden; validation reports and discovery proposals remain
  bounded and do not expose new query endpoints.

Documentation: document table authority, migration/version behavior,
least-privilege account setup, coordinated offline backup/restore, supported
upgrade and rollback, and the absence of automatic file-profile migration.

Environment verification:
- Local: full Go tests and Compose-backed synthetic draft persistence across a
  MariaDB restart passed; runtime and migration grants were inspected.
- Test: concurrent activation, disconnect and ambiguous-commit integration
  cases using isolated database/volume resources remain outstanding.
- Acceptance: verify the exact candidate and both persistence authorities
  only if a separate acceptance environment is provisioned.
- Production promotion: not authorized.

Dependencies: ADR-0006; existing metadata and validation ports.

Definition of Done: contract and integration tests plus the change workflow,
with no production claim.

### BL-023.9: Add the explicit container-network listener

Type: Technical enabler
Status: In Progress
Priority: P1

Problem and scope: add an explicit backend container-network mode which may
bind `0.0.0.0` only when enabled; retain the default loopback-only listener,
local grant, bounded inputs, cancellation, redaction and same-origin guard.
Original Host and Origin must be preserved through the Nginx proxy. Never
trust forwarded headers as authorization.

Affected modules and contracts:
- `cmd/mimir/serve.go` — explicit container-network flag, retaining the
  standalone loopback default.
- `internal/httpapi/` — preserve the exact request Host and Origin contract;
  reject forwarded-host/origin substitution and non-loopback browser origins.
- `contracts/openapi/` and `cmd/mimir/*_test.go` — keep API operations and
  request DTOs unchanged.

Acceptance criteria:
- [x] Given standalone defaults, non-loopback listen addresses are rejected;
  given explicit container-network mode, the backend listens on the bridge
  without a published host port.
- [x] Given proxy-preserved loopback Host and matching HTTP Origin, a valid
  mutation succeeds; missing, forged and mismatched origins or hosts fail
  before side effects. CORS and OPTIONS remain unsupported.
- [x] Given a non-loopback origin, when it reaches the proxy, then the backend
  rejects it before side effects.
- [x] Given missing grant, invalid input, cancellation or ambiguous commit,
  existing authorization and failure semantics remain intact.

TDD first:
- First failing tests: default mode rejects non-loopback; explicit container
  mode accepts only the configured container bind; matching proxy-preserved
  Host/Origin succeeds; missing, mismatched, non-loopback, and forged
  forwarded-origin headers are rejected before operations run.
- Additional tests: `OPTIONS` and CORS remain unsupported; authorization,
  body limits, request cancellation, and stable errors remain unchanged.

Compatibility and rollback:
- Existing standalone `mimir serve` invocation and loopback behavior remain
  unchanged. Container mode is explicit and is not a remote/shared-host mode.
- Roll back by returning to the previous backend image and standalone listener;
  do not alter API contract versions or persisted state.

Security and data impact: backend port is internal to the private Compose
network; only the frontend binds host loopback. Same-origin validation remains
an application guard, not a substitute for process/network isolation. Do not
trust `X-Forwarded-Host` or `X-Forwarded-Proto`.

Documentation: describe the explicit opt-in, loopback-only host publication,
proxy header preservation, local grant, and lack of shared-host support.

Environment verification:
- Local: unit tests for listen address and origin parsing.
- Test: synthetic Nginx-to-backend proxy test with valid and hostile headers.
- Acceptance: verify the exact candidate only if a separate environment is
  provisioned.
- Production promotion: not authorized.

Dependencies: ADR-0005 and ADR-0006; existing API contract and HTTP handler.

Definition of Done: security/contract and proxy tests plus the change workflow.

### BL-023.10: Compose the local three-service deployment

Type: Technical enabler
Status: In Progress
Priority: P1

Problem and scope: ADR-0006 resolves the deployment design. Deliver exactly
`frontend` (Angular/Nginx), `backend` (Go) and `mariadb` on a private bridge;
publish only frontend HTTP on host loopback. Provision named MariaDB and
backend volumes, MariaDB healthcheck, and backend readiness/reconnection.
Require environment or ignored `.env` credentials; add the missing `.env`
ignore rule before use. Include a minimal Angular profile-identity view that
proves same-origin API connectivity; full operation journeys remain in
BL-023.4-.6. The minimal shell is part of this deployment item, so it does not
depend on the blocked full metadata journey.

Affected modules and contracts:
- `frontend/` — pinned Angular build source, accessible profile-status shell,
  and production static build served by Nginx.
- `backend/` — multi-stage Go container image, explicit container-network
  listener and MariaDB-backed metadata composition.
- `docker-compose.yml`, `frontend/nginx.conf`, `deploy/mariadb/`,
  `.env.example`, `.gitignore`, `.github/workflows/ci.yml` — three-service
  topology, proxy, database
  initialization/permissions, required secrets, health checks and checks.
- User and operations docs — setup, shutdown, source mount, both volumes,
  credentials, backup/restore, upgrades and troubleshooting.

Acceptance criteria:
- [x] Given clean configuration, Compose starts exactly the three services;
  only the frontend port is host-published and only backend connects to DB.
- [x] Given a restarted database, backend reconnects and
  reports dependency readiness and failures separately from static health;
  uncertain writes are not blindly retried.
- [x] Given a same-origin frontend request, Nginx serves Angular and proxies `/api/`
  same-origin with original Host/Origin; no CORS or backend host port exists.
- [ ] Given offline backup/restore of both volumes, metadata, revision
  references, execution state, audit and data remain consistent; no
  cross-volume atomicity is claimed.
- [x] Given missing credentials, startup fails safely without defaults or
  disclosure; `.env` is ignored and application DB privileges are scoped.

TDD first:
- First failing checks: Compose configuration exposes only frontend on host
  loopback, has exactly three runtime services and named volumes, rejects
  missing credentials, and builds the Angular and Go images.
- Additional tests: MariaDB readiness/restart, backend reconnect/readiness,
  Nginx Host/Origin preservation, frontend-to-API smoke, restart persistence,
  and offline backup/restore of both volumes.

Security and data impact:
- Expose only frontend HTTP on host loopback; do not publish backend or
  MariaDB ports. The backend runtime account is schema-scoped and the
  migration account is not used by request handlers.
- Require non-empty user-supplied credentials; commit placeholders only.
  `.env` must be ignored. Do not echo secrets in build output, startup logs,
  health responses, or errors.
- Mount the developer's `./data/sources` read-only at `/data/sources` in the
  backend container. This is an explicit local input mount, not a browser
  file-system browser or arbitrary volume API.

Compatibility, migration, and rollback:
- Preserve the existing standalone CLI and file-backed profile. No data
  import or automatic transition to MariaDB occurs.
- Container metadata and backend execution/state data are independent
  authorities on separate named volumes. Stop the stack before backing up or
  restoring both; cross-volume atomic live snapshots are not guaranteed.
- Roll back by stopping the stack and restoring a compatible backend/frontend
  image pair plus coordinated offline volumes. Database schema migrations are
  forward-only; do not downgrade a volume in place.

Documentation: include required credential generation/configuration, local
source placement/mount, startup/readiness, shutdown, both volume authorities,
coordinated offline backup/restore, upgrade compatibility and rollback.

Environment verification:
- Local: Compose config/build and synthetic profile-health connectivity;
  synthetic draft persistence across MariaDB restart, same-origin proxy
  mutation, source-mount boundary, and hostile-origin rejection passed.
- Test: isolated MariaDB and named volumes for readiness/restart, proxy,
  grants and offline restore checks remain outstanding.
- Acceptance: verify exact built images and coordinated volumes only if a
  separate target is provisioned.
- Production promotion: not authorized.

Dependencies: ADR-0006 for design; BL-023.8 and BL-023.9. The initial
connectivity shell is delivered in this item; full GUI journey items
BL-023.4-.6 build on its Angular project and proxy but do not block the
container profile.

Definition of Done: the change workflow and all composition, security,
recovery, compatibility and documentation evidence above.
