# ADR-0005: Local GUI and HTTP endpoint boundary

## Status

Accepted by the user's explicit direction on 2026-11-18. This records the
architecture boundary for BL-023. The Go backend and endpoint contract are
implemented in the bounded BL-023.2/3 slice; Angular screens and static assets
remain unimplemented. [ADR-0006](./ADR-0006-local-compose-and-metadata-persistence.md)
amends the serving and listener choices for an explicit container profile;
the existing standalone behavior remains unchanged.

## Context

The implemented local reference profile is exposed through the Go CLI
(`cmd/mimir/main.go`) and use cases in `internal/local/`. It has no Angular
application. When this decision was accepted, no HTTP server or API contract
existed; BL-023.2/3 now implements the constrained local backend. The local profile is single-process
and local-disk only (ADR-0002); use cases already enforce the explicit
`local-user` / `allow-local` grant. The existing stable failure contract is
`mimir.failure/v1` (`internal/domain/failure.go`), including cancellation,
deadline, and ambiguous-commit categories. The technology baseline already
selects synchronous HTTP/JSON and OpenAPI 3.1 for HTTP contracts
(`docs/software/technology-and-delivery.md`).

BL-023 introduces a concrete local GUI consumer, but it does not justify a
hosted service, general-purpose API, new application queries, or changes to
local-profile behavior. The boundary must retain the existing use cases as
authorities and limit local filesystem access.

## Decision

- The GUI is an Angular browser single-page application. ADR-0006 selects
  Nginx-served assets and same-origin API proxying for the container profile;
  it does not authorize an independently hosted remote service.
- Expose synchronous HTTP/JSON endpoints under `/api/v1`, described with
  OpenAPI 3.1, consistent with the accepted technology baseline. Endpoint
  request and response DTOs map only to existing local-profile use cases. This
  does not add run history, record/query APIs, a dispatcher, or other domain
  behavior.
- The backend is a local, single-user process. Standalone mode defaults to
  `127.0.0.1` and rejects non-loopback addresses. Only the explicit
  container-network mode in ADR-0006 may bind `0.0.0.0`, without a host port.
  That mode is not yet implemented. Startup requires
  an explicit `--allow-local`; each protected use-case call receives the
  existing `local-user` actor and `allow-local` grant. Loopback and this grant
  are local-use controls, not hosted identity or multi-user authorization.
  The process is not safe for shared-host or multi-user exposure.
- Do not expand the operating-system support matrix: backend build targets
  remain those in ADR-0001, and Linux/amd64 is the only runtime-verified target
  at present. GUI runtime support on other targets is not established by this
  ADR.
- Configure one fixed state directory at server startup. Supply the source
  root explicitly for each operation that requires it. Metadata and cleaning
  document inputs are bounded inline JSON; endpoints must not accept arbitrary
  server file paths for those documents.
- Do not enable CORS or remote origins. Mutating browser requests require a
  same-origin check. Enforce JSON content-type and operation-specific request
  body byte limits before side effects; the endpoint contract in BL-023.2
  specifies concrete limits. Propagate HTTP request cancellation and
  deadlines through request context to the use cases. A disconnect cannot be
  reported to the disconnected client; it must not trigger a blind retry.
- Return the existing `mimir.failure/v1` envelope for failures, retaining its
  stable category, operation, retryability, and commit-certainty information.
  Use this explicit HTTP status mapping:

  | Failure category | HTTP status |
  | --- | ---: |
  | `invalid_input` | 400 |
  | `unsupported_capability` | 422 |
  | `access_denied` | 403 |
  | `conflict` | 409 |
  | `policy_rejected` | 422 |
  | `unavailable` | 503 |
  | `ambiguous_commit` | 500 |
  | `cancelled` | 408 |
  | `timeout` | 504 |
  | `internal` | 500 |

  The envelope remains authoritative; HTTP status alone must not imply that
  retry is safe. In particular, preserve unknown commit certainty for
  `ambiguous_commit`, do not expose raw adapter errors, and do not automatically
  retry.
- Do not introduce hosted identity, remote origins, a multi-user mode,
  production hosting, filesystem browsing, or queries and domain behavior
  absent from the existing local-profile capability map.

### BL-023.2/3 operation-level contract choices

The following local-first details complete the contract assignment in this
ADR. They are intentionally limited to the currently implemented use cases:

- Publish the contract as `contracts/openapi/local-profile-v1.json`, using the
  JSON representation of OpenAPI 3.1.0. `/api/v1` is the only API prefix.
  Every request supplies `Mimir-Contract-Version: mimir.gui-api/v1`; an
  unsupported or missing version is rejected before invoking a use case.
  Responses repeat the supported contract version.
- Use these routes only: `GET /health`;
  `POST /metadata/validate`; `GET /metadata/validation-reports/{inputDigest}`;
  `POST /metadata/revisions`; `GET /metadata/revisions`;
  `POST /metadata/activate`; `POST /discovery`; `POST /runs`;
  `POST /schedules/evaluate`; `POST /schedules/stop`; `POST /replays`;
  and `POST /cleaning/preview` or `/cleaning/apply`. No other API route or
  query is part of this contract. The two list/report reads map to their
  existing local use cases.
- Apply request-body ceilings of 1 MiB to inline metadata for validation and
  authoring; 1 MiB plus 4 KiB to the activation envelope (with the inline
  metadata itself still limited to 1 MiB); 256 KiB plus 4 KiB to a cleaning
  envelope (with its record still limited to 256 KiB); and 64 KiB to every
  other request. Reject excess bytes before invoking any use case. JSON
  objects reject unknown fields, and only the documented metadata or cleaning
  record fields accept inline documents.
- Contract version and correlation are transport headers. A supplied
  `X-Correlation-ID` is limited to 1–128 ASCII letters, digits, dot,
  underscore, or hyphen; absent IDs are generated by the server and returned
  in that header. The API does not impose a business-operation deadline:
  client disconnect/cancellation and a caller-provided request deadline flow
  through the request context. HTTP header reads are bounded independently.
- In the implemented standalone mode, `--listen` defaults to `127.0.0.1:8787`;
  only IP-literal loopback hosts are accepted (not hostnames whose resolution
  may vary). The future explicit container-network mode is defined in
  ADR-0006. Startup also requires
  `--allow-local` and one absolute `--state-dir`, normalized once and held
  fixed for the process. Source-root inputs remain explicit per applicable
  operation. Mutating POST requests require a valid same-origin `Origin`
  whose host is loopback; CORS and `OPTIONS` are not supported.

These original API/wiring decisions add no persisted representation or domain
operation. ADR-0006 separately selects a container profile and metadata
persistence target; prior CLI behavior remains unchanged.

## Alternatives considered

- A remote or shared-host service: rejected because the local profile has no
  hosted identity or multi-user authorization and its file ownership model is
  single-process/local-disk only.
- A cross-origin GUI: rejected. ADR-0006 selects separate local containers
  behind one browser origin, not independent remote deployments.
- Accept arbitrary server paths for metadata or cleaning inputs: rejected
  because inline bounded JSON keeps those operations from turning browser
  requests into server-side file access.
- Invent history/query, dispatch, or additional use-case endpoints: rejected
  because the current use cases provide no such contracts.
- Define a new error schema or retry policy: rejected because it would
  duplicate or weaken the existing versioned failure contract and commit
  uncertainty.

## Consequences

BL-023.1's architecture decisions are resolved. BL-023.2/3 has specified the
OpenAPI operations, DTOs, per-operation limits, and thin Go HTTP entrypoint
within this boundary. The delivered API does not include Angular static
assets, installation integration, or a new use case.

Both modes are appropriate only for a local single-user environment; neither
must be presented as safe for shared hosts. Existing CLI
behavior and persisted metadata/state formats remain unchanged. The exact
Angular toolchain and frontend build integration are implementation details
for GUI delivery, not prerequisites for defining the Go endpoint boundary.
No new runtime-verified operating-system claim is made beyond the evidence
already recorded in ADR-0001.

## Owner and affected backlog

Decision authority: the user explicitly accepted these bounded decisions.
Architecture documentation and implementation tracking: BL-023 owner.
Affects BL-023.1, BL-023.2, and BL-023.3, and narrows the API deferral in
[ADR-0004](./ADR-0004-delivery-boundary-and-external-adapters.md).
