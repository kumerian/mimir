# Mimir High-Level Implementation Backlog

## Status and scope

This is the canonical high-level delivery map. It is ordered by proposed
dependency, not by a promise that an item is ready for implementation.

Evidence labels used below:

- **Current**: verified in this repository.
- **Required**: mandated by an accepted architecture or workflow document.
- **Proposed**: a recommendation or an unapproved baseline.
- **Open**: requires an explicit decision or ADR.
- **Assumption**: scope used only to define the next validation step.

Each item must be refined before implementation and must apply
[`../implementation/change-workflow.md`](../implementation/change-workflow.md)
in full.

## First vertical slice refinements

The confirmed BL-001–BL-022 scope is refined in one document per parent:

| Parent | Refinement |
| --- | --- |
| BL-001 | [Foundation decisions and ADR set](./BL-001-foundation-decisions.md) |
| BL-002 | [Reproducible build and CI](./BL-002-reproducible-build-and-ci.md) |
| BL-003 | [Stable contracts](./BL-003-stable-contracts.md) |
| BL-004 | [Contract and recovery test harness](./BL-004-reusable-contract-and-recovery-tests.md) |
| BL-005 | [Security and governance](./BL-005-security-and-governance.md) |
| BL-006 | [Versioned metadata contracts](./BL-006-versioned-metadata-contracts.md) |
| BL-007 | [Metadata control plane](./BL-007-metadata-control-plane.md) |
| BL-008 | [Discovery and schema drift](./BL-008-discovery-and-schema-drift.md) |
| BL-009 | [Reference profile decisions](./BL-009-reference-profile.md) |
| BL-010 | [Persistence, audit, telemetry, and secrets](./BL-010-persistence-audit-telemetry-secrets.md) |
| BL-011 | [Scheduling and run planning](./BL-011-scheduling-and-run-planning.md) |
| BL-012 | [Run state and coordinator](./BL-012-run-state-and-coordinator.md) |
| BL-013 | [Reference data adapters](./BL-013-reference-data-adapters.md) |
| BL-014 | [End-to-end reference ingestion](./BL-014-end-to-end-reference-ingestion.md) |
| BL-015 | [Transformation](./BL-015-transformation.md) |
| BL-016 | [Quality disposition](./BL-016-quality-disposition.md) |
| BL-017 | [Explicit cleaning](./BL-017-explicit-cleaning.md) |
| BL-018 | [Replay](./BL-018-replay.md) |
| BL-019 | [Local entrypoints](./BL-019-entrypoints.md) |
| BL-020 | [Artifact promotion](./BL-020-artifact-promotion.md) |
| BL-021 | [Documentation](./BL-021-documentation.md) |
| BL-022 | [External adapters](./BL-022-external-adapters.md) |

## Repository baseline and comparison

- **Current:** the checkout contains a Go module, local CLI, domain and
  capability contracts, metadata/schema validation, planning, local file
  adapters, a bounded ingestion runtime, local state/audit journals, and
  hermetic unit/contract/recovery tests.
- **Current:** transformations and quality dispositions are integrated into
  the bounded run; discovery persists proposals, scheduling persists requests
  without dispatch, cleaning is explicit and separately audited, and replay
  reconciles the original pinned idempotent batch.
- **Required:** `docs/architecture/system-architecture.md` is the accepted
  system baseline. The Go, JSON v1, local file profile, finite rule catalogs,
  and local CLI decisions are recorded in delegated ADRs and await independent
  maintainer review.
- **Open/blocked:** hosted authorization, production-grade secret
  resolution, distributed worker fencing, deployment environments, signing,
  exact-artifact promotion, and a concrete external adapter target remain
  unavailable or intentionally unselected.

## Priority and delivery model

- **P0** is required to establish a safe, reviewable first vertical slice.
- **P1** is required for the documented product capabilities and initial
  operational delivery.
- **P2** adds replaceable integrations and deployment independence after the
  core behavior is proven.

## Delegated implementation status

Status updated 2026-11-18 under explicit user delegation of the open
architecture decisions. ADRs are implementer decisions awaiting independent
maintainer review; this section distinguishes partial implementation from
acceptance. No item is marked Done because independent review and deployed
acceptance evidence are not available in this task.

| Item | Status | Verified implementation / remaining boundary |
| --- | --- | --- |
| BL-001 | In Progress | Delegated language, JSON v1, local authorization, record, recovery, and delivery decisions recorded in ADR-0001–0004; independent maintainer review pending. |
| BL-002 | In Progress | Go module, named required-check failure test, CI matrix/candidate identity, all six local cross-builds, and Linux clean-install smoke pass; vulnerability database DNS, signing, hosted CI and acceptance remain blocked/unavailable. |
| BL-003 | In Progress | Stable failures, typed identifier/attempt/checkpoint invariants, record/policy/rule/secret/capability validation, versioned ports, and local source/destination cause-reference mapping are tested. External adapter failure contracts remain unavailable until adapters are selected. |
| BL-004 | In Progress | Shared source/destination contracts and deterministic local fault injection cover supported commit, quarantine, checkpoint, audit, and terminal-state boundaries; no previous persisted format or distributed backend migration is claimed. |
| BL-005 | In Progress | Local-user gate, effective object/column public/internal ceiling at activation/discovery/run, and redacted denial audit are tested. Unsupported credential references fail closed; resolver composition, hosted identity and production secret systems remain absent. |
| BL-006 | In Progress | Strict JSON v1 is runtime-validated against the pinned embedded Draft 2020-12 schema; deterministic canonicalization/reports, safe diagnostics, duplicate/unknown/version rejection and fixtures pass. The initial v1 format has no prior migration; hosted acceptance remains unavailable. |
| BL-007 | In Progress | File metadata/reports use versioned ports; immutable validation reports are persisted/queryable and never execution-eligible; authoring, conditional activation, audit and policy resolution are tested. Hosted authorization/storage acceptance remains absent. |
| BL-008 | In Progress | Local JSONL discovery provider is capability-checked and persists digest-linked schema proposals without changing active metadata; proposal review/application is manual and only the single-object profile is supported. |
| BL-009 | In Progress | Delegated local profile documents crash outcomes and recovery, commit/checkpoint ordering, tail repair, locks, and permissions; independent approval and production suitability remain pending. |
| BL-010 | In Progress | Metadata, plan, fenced run-state/checkpoint, data, quarantine/cleaned output, required audit, and best-effort correlated telemetry are separate; metrics/traces exporters, resolver composition, retention policy, and hosted secrets remain absent. |
| BL-011 | In Progress | Capability-matched immutable plans, fixed-clock schedule slots and stable requests exist; cancellation/shutdown now writes immutable stop acknowledgements/audit without changing run state. Requests do not dispatch; external scheduler acceptance remains absent. |
| BL-012 | In Progress | File run-state store conditionally appends owner-fenced attempt history and failure stage; local cancellation, checkpoint reconciliation, and supported-boundary crash matrix pass. Distributed leases and worker shutdown protocol remain absent. |
| BL-013 | In Progress | Local JSONL source/destination ports pass shared contracts, duplicate/reconciliation checks, and timeout/cancellation/denied-scope tests; external adapter certification remains unavailable. |
| BL-014 | In Progress | Synthetic CLI runs integrate transformations, quality, quarantine, state, audit, and telemetry; local durable/replay fault matrices and clean install smoke pass on Linux/amd64. Local source/destination capability and basic configuration preflight is tested; full composition checks, other native OS smokes, acceptance, and exact-candidate evidence remain outstanding. |
| BL-015 | In Progress | Ordered `v1` transformations are pinned in plans and integrated with ingestion/provenance; deployed acceptance remains untested. |
| BL-016 | In Progress | `v1` quality rules, warning/reject/quarantine integration, durable quarantine and recovery are implemented; automatic retention and hosted policy are not provided. |
| BL-017 | In Progress | Explicit `v1` cleaning has preview, local access/classification checks, separate idempotent persistence, redacted audit and interruption recovery; no hosted authorization or deployed acceptance. |
| BL-018 | In Progress | CLI replay validates pinned metadata/source/rules/adapter inputs and reconciles the original batch; local replay and run fault matrices now verify restart idempotency. New output-producing replay and external-adapter acceptance remain out of scope. |
| BL-019 | In Progress | Health, validate, author/list, activate, discovery, schedule, run, clean and replay CLI commands exist; HTTP API, worker, hosted identity and contract-test host remain. |
| BL-020 | Blocked | Candidate identity/checksum exists. Test/acceptance/production deployment target, human approval configuration, signing, smoke and rollback drill are absent. Next action: operators provision and protect the required environments and signing identity. |
| BL-021 | In Progress | README and local guides cover implemented behavior; Linux/amd64 clean install and offline whole-directory backup/restore drill pass. Other-target installs, migration and candidate promotion/rollback remain unverified or externally blocked. |
| BL-022 | Blocked | ADR-0004 explicitly defers external adapters because no concrete integration need or target is evidenced. Next action: a concrete integration requirement with an approved adapter ADR and isolated test resources. |

The proposed increments follow the accepted system delivery order: contracts,
metadata, planning, runtime, safe reference data movement, transformation,
quality, explicit cleaning and replay, then additional adapters and
independent entrypoints.

## Blocker register

| Blocker | Decision owner | Affects | Resolution item |
| --- | --- | --- | --- |
| Independent maintainer review of delegated ADR-0001–0004 and local reference implementation | Maintainers/architecture authority | BL-001 and dependent review/acceptance states | Review this change before promotion |
| Hosted identity, externally enforced activation approval, and production secret-provider contract | Security/architecture authority | Hosted BL-005, BL-019, BL-020 | Refine when a hosted target is selected |
| Distributed worker fencing and worker shutdown protocol | Runtime/operations owner | BL-012, BL-014 | Select and provision a multi-worker state-store/deployment profile; local single-host fencing and recovery are implemented |
| Test/acceptance/production targets, artifact signing, protected approvals, and rollback drill | Repository administrators/environment operators | BL-020 | Provision environments and signing identity |
| Concrete external adapter use case and isolated test resources | Product/architecture/operations owner | BL-022 | Submit evidence-backed integration request |
| Production retention, data access, and smoke policies | Governance/operations owner | BL-005, BL-020–BL-022 | Establish with the production service owner |

## Delivery map

### BL-001: Approve foundation decisions and ADR set

Type: Architecture decision  
Status: In Progress
Priority: P0

Problem/outcome:
The initial implementation needed explicit decisions before local behavior
could proceed. The user delegated those choices to the implementer; ADR-0001
through ADR-0004 record the conservative decisions and their limits. Independent
maintainer review is still required before promotion.

Scope:
- approve or revise the proposed Go/module boundary and the initial one-process
  delivery model;
- decide the supported Go floor, operating-system/architecture matrix, pinned
  task runner, linter, JSON Schema, and OpenAPI tooling;
- define contract and artifact version compatibility, unsupported-version
  behavior, migration policy, and rollback expectations;
- define exact port method/error shapes, canonical record contract, and the
  boundary for adapter-owned binary or streaming transfer representations;
- define authorization inputs, activation approval, retention/data classes,
  service-level objectives, and the single-/multi-binary API and worker shape;
- resolve which remaining choices are prerequisites for each dependent item
  and record accepted alternatives and consequences as ADRs. Hosted products
  and production protocols remain deferred until a concrete target exists.

Out of scope:
- selecting a database, cloud, orchestrator, message broker, or deployment
  product without separate evidence and approval;
- implementing adapters or runtime behavior.

Dependencies:
- None.

Architecture traceability:
- `docs/architecture/system-architecture.md` — Open Decisions and Architecture Invariants.
- `docs/architecture/software-architecture.md` — Open Decisions and Repository/Build shape.
- `docs/software/technology-and-delivery.md` — Technology Decision and Open Decisions.
- `docs/implementation/change-workflow.md` — Impact and Architecture Review.

Acceptance outcomes:
- [x] Each open decision has either a delegated ADR, an explicit later
  decision gate, or a documented out-of-scope rationale.
- [x] BL-001 records which proposed documents and choices are accepted for
  this implementation; independent maintainer review remains pending.
- [x] Decisions preserve platform neutrality, immutable plans,
  adapter substitutability, safe recovery, and the one-artifact promotion model.
- [x] Dependent backlog items identify which ADR gates their implementation.

Required evidence:
- Tests: Not applicable to decisions; validate ADR/links with documentation checks.
- Documentation: ADRs and approved baseline updates in the affected documents.
- Installation/operations: supported-platform and release-identity constraints recorded.
- Environments: local, test, acceptance, and production implications recorded per decision.

Open decisions or assumptions:
- **Pending review:** the delegated ADR set awaits independent maintainer
  review; the user-authorized implementation decisions are recorded as
  accepted for this local delivery.
- **Later gates:** hosted authorization, production retention/SLO policy,
  API/worker protocol, signing service, deployment/rollback platform, and
  external adapter product remain unselected.

### BL-002: Establish reproducible Go build, local workflow, and core CI

Type: Technical enabler  
Status: In Progress
Priority: P0

Problem/outcome:
A clean checkout must provide a pinned, hermetic, cross-platform path to
format, lint, test, generate, build, document, and package Mimir before
behavioral increments are added.

Scope:
- establish the approved Go module/workspace, repository layout, tool pins,
  dependency policy, and task definitions;
- provide `test`, `test:race`, `test:contract`, `test:int`, `lint`, `build`,
  `package`, and `docs` task surfaces;
- configure fast hermetic default tests and pinned formatting, vet, lint,
  static analysis, vulnerability, and dependency checks;
- add GitHub Actions pull-request and merge verification with explicit
  passed/failed/not-applicable/blocked outcomes;
- provide non-secret local configuration and clean-checkout developer and
  contributor documentation.

Out of scope:
- product use cases, production adapters, acceptance topology, or release
  promotion.

Dependencies:
- BL-001.

Architecture traceability:
- `docs/architecture/software-architecture.md` — Software Style, Logical Module Structure, Testing Architecture.
- `docs/software/technology-and-delivery.md` — Repository and Build Layout, Development Toolchain, Local Development, CI Pipeline.
- `docs/implementation/change-workflow.md` — TDD and Verification and CI/CD.

Acceptance outcomes:
- [ ] A clean checkout restores pinned tools and completes format, lint, vet,
  dependency, documentation, and hermetic unit checks.
- [ ] CI reports affected and not-applicable checks explicitly and cannot hide
  a failing required check.
- [ ] Supported-platform binaries and reproducible candidate artifacts are
  buildable from the same reviewed source revision.
- [x] Local documentation requires no production credentials or undocumented global tool.

Required evidence:
- Tests: repository smoke, task smoke, tool-pin, generated-artifact freshness, and dependency-policy checks.
- Documentation: contributor setup, commands, troubleshooting, and package output.
- Installation/operations: clean install from release archive/container where applicable.
- Environments: local and isolated test CI; acceptance packaging inputs defined.

Open decisions or assumptions:
- Go/module and target matrix are selected for this local delivery. A separate
  configured linter, semantic JSON Schema generator, OpenAPI tooling, and
  artifact-signing platform remain later gates; no API currently exists.

### BL-003: Deliver stable domain, capability, error, and port contracts

Type: Technical enabler  
Status: In Progress
Priority: P0

Problem/outcome:
All modules need a technology-independent shared language, explicit adapter
capabilities, and stable failure behavior before they can implement use cases.

Scope:
- implement opaque identifiers and immutable domain values for metadata
  revisions, policies, run plans, runs, attempts, checkpoints, records, batches,
  lineage, rules, adapters, and secret references;
- implement core domain invariants and stable error categories, including
  retryability, commit certainty, reconciliation action, stage, and adapter
  cause reference;
- define versioned capabilities and port contracts for repositories,
  discovery, sources, destinations, scheduling, state, secrets, engines,
  audit, and telemetry;
- enforce dependency direction so domain/application/ports do not depend on
  adapters or entrypoints.

Out of scope:
- adapter implementations, persistence behavior, and executable product use cases.

Dependencies:
- BL-001.
- BL-002.

Architecture traceability:
- `docs/architecture/system-architecture.md` — Domain Contracts, Adapter Ports, Failure Model, Dependency Rules.
- `docs/architecture/software-architecture.md` — Domain Layer, Port Contracts, Adapter Capabilities, Error Contract.
- `docs/software/technology-and-delivery.md` — Module Technology Map.

Acceptance outcomes:
- [ ] Domain construction and state invariants are table-tested, and domain
  packages have no infrastructure imports.
- [ ] Each port returns domain values and stable Mimir errors rather than
  implementation-owned exceptions.
- [ ] Every capability declares version, operations, guarantees, limits, and
  explicit unsupported behavior.
- [ ] Public contracts serialize or reject versions according to the approved
  compatibility policy without partial side effects.

Required evidence:
- Tests: unit, boundary, property-based where useful, version/unknown-field, and dependency tests.
- Documentation: contract reference, capability semantics, and stable error catalog.
- Installation/operations: report supported contract versions in builds/health later through BL-019.
- Environments: all contract tests run hermetically in local and test.

Open decisions or assumptions:
- **Open:** exact method signatures, error serialization, canonical record
  contract, and version-negotiation policy.
- **Assumption:** JSON is used only at external/metadata boundaries as
  proposed in the technology baseline; it does not predetermine persisted data formats.

### BL-004: Create reusable contract, dependency, compatibility, and recovery tests

Type: Technical enabler  
Status: In Progress
Priority: P0

Problem/outcome:
Mimir requires repeatable proof that adapters are substitutable, module
boundaries hold, versions migrate safely, and execution recovers at every
durable boundary.

Scope:
- create reusable public contract fixtures for repositories, schedulers,
  engines, observability, secret resolvers, sources, destinations, and state stores;
- implement architecture dependency and version-compatibility test facilities;
- implement deterministic synthetic data, fault injection, cancellation,
  timeout, duplicate-delivery, stale-worker, and crash boundary fixtures;
- integrate unit, contract, integration, race, and acceptance classifications
  into the build from BL-002.

Out of scope:
- product-specific adapter implementation and deployed load testing.

Dependencies:
- BL-003.

Architecture traceability:
- `docs/architecture/software-architecture.md` — Testing Architecture and Suggested Package Acceptance Criteria.
- `docs/software/technology-and-delivery.md` — Testing Strategy, Test Classification, Test Data.
- `docs/implementation/change-workflow.md` — Required Tests by Change Type.

Acceptance outcomes:
- [ ] A conforming adapter must pass the same externally observable suite used
  by reference implementations.
- [ ] Contract rejection, cancellation, timeout, duplicate delivery, stale
  ownership, and ambiguous-commit scenarios assert observable state.
- [ ] Previous supported contracts can be read, migrated, rejected, or rolled
  back according to BL-001's compatibility policy.
- [ ] Dependency tests fail if a domain/application/ports package imports infrastructure.

Required evidence:
- Tests: the test infrastructure itself has focused unit tests and fixture validation.
- Documentation: external contract-test consumer guide and test-class command reference.
- Installation/operations: clean ephemeral test setup and cleanup.
- Environments: local and test always; acceptance fixtures reused where safe.

Open decisions or assumptions:
- **Open:** supported previous contract versions and exact race/recovery fixture boundaries.
- **Assumption:** synthetic data is the default and production-derived fixtures require separate approval.

### BL-005: Enforce secret-reference, authorization, classification, and redaction policy

Type: Technical enabler  
Status: In Progress
Priority: P0

Problem/outcome:
Metadata, planning, and runtime must enforce least-privilege access and data
governance without embedding secrets or leaking sensitive values.

Scope:
- implement opaque secret-reference values and short-lived resolver interaction contracts;
- define authorization contexts and decision points for metadata authoring,
  discovery, activation, planning, and stage execution;
- represent retention and data-classification constraints for sources, objects,
  columns, rules, destinations, and telemetry;
- enforce redaction and denial behavior in errors, plans, logs, traces, audit,
  lineage, and diagnostics;
- define and test denied access, unsupported references, and policy failures
  without success-shaped behavior.

Out of scope:
- selecting an identity provider or concrete secret store;
- permissive default access to simplify development.

Dependencies:
- BL-003.
- BL-004.

Architecture traceability:
- `docs/architecture/system-architecture.md` — Security and Governance.
- `docs/architecture/software-architecture.md` — Security Boundaries.
- `docs/implementation/change-workflow.md` — Security, Privacy, and Compliance.

Acceptance outcomes:
- [ ] Secret values never enter metadata, plans, test output, logs, lineage, or audit records.
- [ ] A denied metadata, adapter, destination, or rule operation creates no
  partial authorization outcome and emits a redacted audit category.
- [ ] Classification policy can block unsuitable destinations, transformations,
  cleaning, observability payloads, and retention.
- [ ] Resolver and policy adapters expose capability, timeout, cancellation,
  denied-reference, and redaction contract coverage.

Required evidence:
- Tests: denied/allowed policy, least privilege, timeout, cancellation, redaction, and audit tests.
- Documentation: permission matrix, secret-reference rules, classification and retention behavior.
- Installation/operations: secret-reference configuration and no-secret troubleshooting guidance.
- Environments: development resolver locally; isolated policy tests and independently managed secrets later.

Open decisions or assumptions:
- **Open:** authorization protocol, identity provider, policy-evaluation
  mechanism, retention periods, and data-classification vocabulary.
- **Assumption:** development configuration may use synthetic references only.

### BL-006: Establish versioned metadata contracts and schema validation

Type: Technical enabler  
Status: In Progress
Priority: P0

Problem/outcome:
Mimir cannot author, validate, or activate safe metadata until its versioned
JSON/YAML authoring boundaries and domain model have deterministic semantics.

Scope:
- model sources, objects, columns, mappings, schedules, transformation and
  quality/cleaning rules, governance, secret references, and metadata revisions;
- define JSON contracts and JSON Schema validation, with human YAML converted
  to canonical JSON and duplicate/unknown-field handling;
- model validation results for structural, semantic, compatibility, and
  security failures;
- define append-only revision identity and compatibility/migration fixtures.

Out of scope:
- persistence implementation, activation workflow, schema observation, and data execution.

Dependencies:
- BL-001.
- BL-003.
- BL-004.
- BL-005.

Architecture traceability:
- `docs/architecture/system-architecture.md` — Domain Contracts and Control Plane.
- `docs/architecture/software-architecture.md` — Domain Aggregates and Metadata Use Cases.
- `docs/software/technology-and-delivery.md` — Metadata domain/policy and Record and Metadata Representation.

Acceptance outcomes:
- [ ] Valid, invalid, unknown-field, duplicate-key, version, migration, and
  unsupported-version fixtures produce deterministic results.
- [ ] A metadata validation result identifies field, rule, affected scope,
  security impact, and compatibility decision.
- [ ] Domain revisions preserve immutable identity and serialize canonically at
  the approved boundary.
- [ ] Draft metadata carries no execution authority.

Required evidence:
- Tests: schema fixtures, round trips, invariants, unknown fields, versions, and migration.
- Documentation: schema reference, authoring examples, constraints, and compatibility limits.
- Installation/operations: schema/tool distribution and validation command.
- Environments: hermetic local/test; acceptance contract compatibility checks.

Open decisions or assumptions:
- **Open:** exact compatibility policy, migration mechanism, activation
  approval representation, and full canonical domain type semantics.
- **Assumption:** JSON/YAML and JSON Schema remain in scope if BL-001 approves the proposed technology baseline.

### BL-007: Deliver metadata authoring, validation, activation, and policy resolution

Type: Feature  
Status: In Progress
Priority: P0

Problem/outcome:
Users and operators need a complete control-plane use case that accepts
candidate metadata but permits execution only after deterministic validation,
authorization, activation, and policy resolution.

Scope:
- implement candidate authoring, validation reports, append-only repository
  port behavior, conditional activation, supersession, and audit;
- implement deterministic inheritance and effective-policy resolution from
  source to object to column and rule while retaining value origins;
- pin active revision lookup and prevent draft or unvalidated metadata from
  authorizing runs;
- provide in-memory/repository test doubles and a replaceable metadata
  repository contract implementation for local use.

Out of scope:
- production persistence choice, discovery, scheduling, or ingestion.

Dependencies:
- BL-004.
- BL-005.
- BL-006.

Architecture traceability:
- `docs/architecture/system-architecture.md` — Control Plane and Architecture Invariants 1-3.
- `docs/architecture/software-architecture.md` — Metadata Use Cases and Metadata Repository.
- `docs/implementation/change-workflow.md` — State/data compatibility and audit requirements.

Acceptance outcomes:
- [ ] An invalid, unauthorized, stale, or superseded revision cannot become active.
- [ ] Two activation requests cannot leave incompatible active revisions.
- [ ] A resolved effective policy deterministically identifies the origin of each override.
- [ ] Author, validate, activate, access, and supersede outcomes emit durable audit intents and support replay of their history.

Required evidence:
- Tests: application use cases, invalid/valid/boundary policies, activation races, audit loss, and previous-version fixtures.
- Documentation: authoring, validation, activation, supersession, and recovery guide.
- Installation/operations: local non-production setup and metadata backup/restore requirements.
- Environments: local in-memory/test double; test contract; acceptance promotion later.

Open decisions or assumptions:
- **Open:** activation approval mechanism and durable audit implementation.
- **Assumption:** a local in-repository repository double is development-only and not the production persistence decision.

### BL-008: Deliver discovery and controlled schema-drift proposals

Type: Feature  
Status: In Progress
Priority: P1

Problem/outcome:
Observed source structure must be normalized and proposed as versioned metadata
without silently rewriting or broadening an active contract.

Scope:
- implement discovery through a declared-capability provider port;
- normalize observed schema into domain observations and preserve provenance;
- classify drift, downstream compatibility, and affected scope;
- apply reject, hold, quarantine, or continue policy as metadata, never by
  implicit active-state mutation;
- submit proposed revisions for the BL-007 validation and activation path.

Out of scope:
- a concrete source technology, automatic activation, or inferred data correction.

Dependencies:
- BL-004.
- BL-007.

Architecture traceability:
- `docs/architecture/system-architecture.md` — Control Plane and Schema Drift.
- `docs/architecture/software-architecture.md` — Discover Metadata.
- `docs/software/technology-and-delivery.md` — Metadata/discovery use cases and contract scenarios.

Acceptance outcomes:
- [ ] Discovery records an observation and proposal without changing the active revision.
- [ ] Reject, hold, quarantine, and continue policies are explicit, auditable, and deterministic.
- [ ] Compatibility failure blocks proposal activation with a reason and affected scope.
- [ ] A provider with missing capability is rejected before source inspection side effects.

Required evidence:
- Tests: discovery contract, drift classifications, policy fixtures, partial observations, and activation race.
- Documentation: discovery permissions, drift review, examples, and recovery.
- Installation/operations: provider registration and isolated discovery configuration.
- Environments: deterministic local provider, contract test, and acceptance-like schema drift.

Open decisions or assumptions:
- **Open:** initial discovery-provider technologies and review granularity.
- **Assumption:** proposals do not auto-activate regardless of provider behavior.

### BL-009: Approve first reference-adapter and persistence profile

Type: Architecture decision  
Status: In Progress
Priority: P0

Problem/outcome:
Recovery and end-to-end behavior require a minimum concrete capability set and
reference implementation profile, but the accepted architecture leaves those
technologies and guarantees open.

Scope:
- choose the minimum reference source and destination capabilities, including
  batch scope, position, idempotency, commit, acknowledgement, and reconciliation;
- choose local/reference metadata, run-state, audit-journal/outbox, and
  development data persistence implementations;
- define checkpoint granularity, state/concurrency primitive, state format,
  event/request envelope, and supported replay limits;
- define the first supported delivery guarantee and explicitly rejected source/
  destination combinations;
- record non-reference extension requirements and migration/rollback behavior.

Out of scope:
- building the selected implementations and selecting production-hosted infrastructure for unrelated environments.

Dependencies:
- BL-001.
- BL-007.
- BL-008.

Architecture traceability:
- `docs/architecture/system-architecture.md` — Idempotency and Recovery, Open Decisions.
- `docs/architecture/software-architecture.md` — Persistence Strategy, Execution Model, Open Decisions.
- `docs/software/technology-and-delivery.md` — Interaction Model and Open Decisions.

Acceptance outcomes:
- [ ] The profile names capability guarantees rather than only product names.
- [ ] A safe commit/checkpoint/source-acknowledgement sequence is specified and recoverable at each boundary.
- [ ] Reference choices have alternatives, consequences, migration, rollback,
  security, and operational limits in ADRs.
- [ ] Every dependent product remains replaceable through BL-003 ports.

Required evidence:
- Tests: not applicable to the decision; acceptance scenarios are specified for later implementation.
- Documentation: ADRs and capability matrix for the reference profile.
- Installation/operations: local resource, backup, restore, cleanup, and limits.
- Environments: local/test requirements and production suitability constraints.

Open decisions or assumptions:
- **Open:** reference source, destination, metadata repository, state format,
  persistence, audit journal/outbox, checkpoint model, and concurrency mechanism.
- **Assumption:** local reference technology is not presumed suitable for production.

### BL-010: Provide persistence, audit, telemetry, and secret-reference adapters

Type: Technical enabler  
Status: In Progress
Priority: P0

Problem/outcome:
Control-plane and runtime outcomes require durable state, auditable decisions,
correlated telemetry, and bounded secret resolution without coupling the core
to a selected backend.

Scope:
- implement BL-009's approved metadata, state, audit, telemetry, and development
  secret resolver adapters with shared port contract coverage;
- keep metadata revisions, execution state, ingested data, audit, and telemetry
  logically separate even when storage shares a backing technology;
- implement correlation propagation and redaction for logs, metrics, traces,
  audit, and lineage;
- implement durable audit/outbox or retryable local journal semantics around
  required business outcomes;
- document retention, backup, migration, and failure/export behavior.

Out of scope:
- source/destination data movement, production platform deployment, and
  silently lost required audit.

Dependencies:
- BL-004.
- BL-005.
- BL-007.
- BL-009.

Architecture traceability:
- `docs/architecture/system-architecture.md` — State, Data, and Audit Stores and Observability.
- `docs/architecture/software-architecture.md` — Persistence Strategy.
- `docs/software/technology-and-delivery.md` — Observability adapters and Environment Configuration.

Acceptance outcomes:
- [ ] Required audit outcomes remain recoverable when telemetry export or the
  audit sink is temporarily unavailable.
- [ ] A metadata, state, audit, or secret adapter passes its shared contract,
  cancellation, timeout, redaction, and integration suites.
- [ ] Logs, metrics, traces, lineage, and audit correlate through revision, plan,
  run, attempt, stage, operation, and record identity where applicable.
- [ ] Operational sampling never drops activation, access, cleaning, or ambiguous-commit audit evidence.

Required evidence:
- Tests: adapter contract, audit durability, migration/rollback, correlation, redaction, and export failure.
- Documentation: adapter capabilities, retention, troubleshooting, backup, restore, and migration.
- Installation/operations: ephemeral test dependencies, cleanup, health, and least privilege.
- Environments: local, isolated test, acceptance-like durability, production capacity gates.

Open decisions or assumptions:
- **Open:** concrete adapter products, durable audit mechanism, retention and sampling policies.
- **Assumption:** operational telemetry may be best-effort only where policy says it is not audit evidence.

### BL-011: Deliver scheduling, capability matching, and immutable run planning

Type: Feature  
Status: In Progress
Priority: P0

Problem/outcome:
An eligible trigger must produce an immutable, safe, replayable run plan without
allowing a scheduler to own ingestion logic or execution state.

Scope:
- implement in-process schedule evaluation, dependency, retry, concurrency,
  scope, and activation eligibility;
- implement `PlanRun` and `ReplayRun` preparation using a pinned active metadata
  revision and resolved policy;
- match source, destination, engine, checkpoint, commit, and security
  capabilities before planning;
- reject unsupported, changed, unsafe, unavailable, or unauthorized combinations
  without side effects;
- persist planned run identity and deterministic plan inputs.

Out of scope:
- external orchestrator implementation, stage execution, and retries that are unsafe without checkpoints.

Dependencies:
- BL-005.
- BL-007.
- BL-009.
- BL-010.

Architecture traceability:
- `docs/architecture/system-architecture.md` — Scheduling and Planning and Deterministic Replay.
- `docs/architecture/software-architecture.md` — Planning Use Cases and Concurrency and Scheduling.
- `docs/software/technology-and-delivery.md` — Scheduling adapter and initial in-process behavior.

Acceptance outcomes:
- [ ] Only validated active metadata and authorized scope can produce a run plan.
- [ ] The immutable plan identifies revision, source/destination capabilities,
  stages, strategy, scope, rule/adapter versions, and safe delivery behavior.
- [ ] Missing or changed required capability is rejected before runtime side effects.
- [ ] Duplicate trigger delivery is idempotent by stable request/run identity.

Required evidence:
- Tests: dependency, trigger, cancellation, timeout, duplicate delivery, capability rejection, policy denial, replay preparation, and plan immutability.
- Documentation: schedule semantics, planner errors, replay limitations, and examples.
- Installation/operations: in-process scheduler configuration, limits, and shutdown.
- Environments: hermetic local/test and acceptance trigger/load scenarios.

Open decisions or assumptions:
- **Open:** service-level expectations, external request envelope, replay-policy detail, and initial external scheduler choice.
- **Assumption:** the first scheduler is in-process as required by the accepted system delivery sequence.

### BL-012: Deliver run, attempt, checkpoint, fencing, and coordinator semantics

Type: Technical enabler  
Status: In Progress
Priority: P0

Problem/outcome:
Execution needs a single-writer, recoverable state model that prevents stale
workers and duplicate attempts from overwriting newer evidence.

Scope:
- implement run/attempt states, transition policy, immutable history, pause,
  resume, retry, cancellation, and terminal outcomes;
- acquire and fence execution ownership through the state-store port;
- implement stage coordination, stable stage commands/results, checkpoint
  validity, failure history, reconciliation, graceful shutdown, and bounded
  in-flight behavior;
- preserve state separately from ingested data and pin a plan throughout execution.

Out of scope:
- concrete source/destination publication, transformation rules, or data-quality behavior.

Dependencies:
- BL-004.
- BL-009.
- BL-010.
- BL-011.

Architecture traceability:
- `docs/architecture/system-architecture.md` — Core Execution Semantics and Idempotency and Recovery.
- `docs/architecture/software-architecture.md` — Run Coordinator, Publication and Checkpoint Ordering.
- `docs/software/technology-and-delivery.md` — Run and attempt state and module recovery tests.

Acceptance outcomes:
- [ ] Every invalid transition and stale worker is rejected without overwriting newer evidence.
- [ ] Retry creates a new attempt or explicit replay and retains prior failure and commit evidence.
- [ ] Checkpoints are accepted only with approved durability evidence and safe resume semantics.
- [ ] Cancellation, timeout, shutdown, duplicate delivery, and interrupted paths reach deterministic states.

Required evidence:
- Tests: state machine, race/fencing, crash around every durable boundary, duplicate delivery, cancellation, timeout, and recovery.
- Documentation: lifecycle, checkpoint, ownership, failure, retry, cancellation, and reconciliation.
- Installation/operations: bounded worker limits, shutdown, state backup, and startup validation.
- Environments: local/test fault injection, acceptance recovery, and production restricted fencing smoke.

Open decisions or assumptions:
- **Open:** concurrency primitive, checkpoint format/granularity, and allowed resume strategies.
- **Assumption:** no exactly-once guarantee is inferred from the state store alone.

### BL-013: Deliver source and destination reference adapters with safe commit contracts

Type: Feature  
Status: In Progress
Priority: P0

Problem/outcome:
The first ingestion path needs interchangeable, capability-declared adapters
that safely stage, commit, deduplicate, and reconcile destination work.

Scope:
- implement the BL-009 reference batch source and destination adapters;
- declare scope, position, batching, acknowledgement, schema observation,
  idempotency, commit, classification, limits, and unsupported operations;
- implement open/read, stable identities, staged writes, durable commit evidence,
  reconciliation, cancellation, timeout, and stable error translation;
- publish reusable conformance and backend-specific integration suites.

Out of scope:
- the planner/coordinator, normalization, transformations, quality, cleaning, and all unapproved platform adapters.

Dependencies:
- BL-003.
- BL-004.
- BL-009.
- BL-011.
- BL-012.

Architecture traceability:
- `docs/architecture/system-architecture.md` — Adapter Ports and Idempotency and Recovery.
- `docs/architecture/software-architecture.md` — Source/Destination ports and Publication and Checkpoint Ordering.
- `docs/software/technology-and-delivery.md` — Source/Destination Adapter test requirements.

Acceptance outcomes:
- [ ] Every adapter declares all supported and omitted capabilities and contract versions.
- [ ] Repeated batch/record delivery produces no duplicate committed result under the approved strategy.
- [ ] An uncertain commit is reconciled and not blindly retried.
- [ ] Secret, classification, cancellation, timeout, and adapter error behavior passes the shared suite.

Required evidence:
- Tests: shared contract, duplicate write, crash before/after commit, capability rejection, cancellation, timeout, and integration.
- Documentation: capabilities, guarantees, limitations, setup, cleanup, and reconciliation.
- Installation/operations: local adapter setup without production credentials and backend-specific diagnostics.
- Environments: local deterministic, ephemeral test, acceptance-like failure scenarios.

Open decisions or assumptions:
- **Open:** reference source/destination products and first delivery/capability profile.
- **Assumption:** the initial path is bounded batch; streaming support is not implied.

### BL-014: Deliver an end-to-end safe reference ingestion path

Type: Feature  
Status: In Progress
Priority: P0

Problem/outcome:
Mimir needs a runnable vertical slice that proves validated metadata, planning,
source read, destination publication, checkpoint, state, audit, and recovery
work together in one process.

Scope:
- compose metadata control plane, in-process scheduler, planner, run coordinator,
  state/audit/telemetry, and reference adapters;
- execute a bounded raw-ingestion path while retaining source identity,
  provenance, and stage correlation;
- apply commit-before-checkpoint-before-acknowledgement ordering where supported;
- provide a local CLI/local entry surface, health check, and deterministic
  synthetic acceptance scenario.

Out of scope:
- transformations, quality, explicit cleaning, public API, external worker
  deployment, and production-scale adapters.

Dependencies:
- BL-010.
- BL-011.
- BL-012.
- BL-013.

Architecture traceability:
- `docs/architecture/system-architecture.md` — Execution Plane, Initial Delivery Shape.
- `docs/architecture/software-architecture.md` — Execution Use Cases and Initial Implementation Increment 5.
- `docs/software/technology-and-delivery.md` — Local Development and production-smoke model.

Acceptance outcomes:
- [ ] A synthetic run completes and exposes matching state, destination, audit,
  lineage, and telemetry identities.
- [ ] Restarting at supported crash boundaries resumes, deduplicates, or
  reconciles without success-shaped ambiguity.
- [ ] Invalid metadata, missing capability, unauthorized destination, and
  uncertain commit prevent unsafe publication.
- [ ] A clean local installation can run and clean up the scenario without production accounts.

Required evidence:
- Tests: end-to-end fault injection, duplicate delivery, resume, cancellation, local smoke, and acceptance scenario.
- Documentation: first-run functional guide, prerequisites, outputs, failures, recovery, and limitations.
- Installation/operations: local install/configure/run/cleanup/health and backup/rollback notes.
- Environments: local, test, acceptance; production only restricted smoke after BL-020.

Open decisions or assumptions:
- **Open:** whether the first public local command is the eventual CLI surface and the approved reference adapters.
- **Assumption:** this slice runs in one process and uses only the approved reference profile.

### BL-015: Deliver normalization and versioned transformation

Type: Feature  
Status: In Progress
Priority: P1

Problem/outcome:
Ingested records must become canonical, source-agnostic representations through
explicit, ordered, deterministic, versioned rules with traceable provenance.

Scope:
- implement canonical normalized and transformed record representations;
- define rule ordering, metadata pinning, idempotency, deterministic errors, and
  rule capability contracts;
- implement normalization such as approved type, naming, null, unit, mapping,
  and enrichment behavior as versioned rules;
- retain source/normalized/transformed provenance references without duplicating raw sensitive values;
- preserve adapter substitutability for reference and external engines.

Out of scope:
- silent inferred transformation, quality disposition, data cleaning, and an unapproved external transformation language.

Dependencies:
- BL-001.
- BL-003.
- BL-004.
- BL-014.

Architecture traceability:
- `docs/architecture/system-architecture.md` — Execution Plane and Quality and Cleaning distinction.
- `docs/architecture/software-architecture.md` — Data Flow and Transformation Engine port.
- `docs/software/technology-and-delivery.md` — Transformation engine test approach.

Acceptance outcomes:
- [ ] Equal pinned inputs, rule order, versions, and adapter contracts produce
  byte-/value-stable approved canonical output.
- [ ] Rule failure identifies input operation, rule version, stage, retryability,
  and whether partial commit exists.
- [ ] Transformation does not access platform clients directly or mutate preserved input provenance.
- [ ] Unsupported canonical representation or rule capability is rejected during planning.

Required evidence:
- Tests: golden, determinism, idempotency, version mismatch, invalid rule, cancellation, timeout, and provenance.
- Documentation: rule authoring/order, canonical semantics, limits, failures, and synthetic examples.
- Installation/operations: engine registration, resource limits, and version compatibility.
- Environments: local/test and acceptance representative transformations; production smoke after promotion.

Open decisions or assumptions:
- **Open:** full canonical binary/stream representation and the approved first reference rule set.
- **Assumption:** the core rule engine is Go as proposed, while external languages remain behind versioned adapters.

### BL-016: Deliver quality evaluation, quarantine, and disposition

Type: Feature  
Status: In Progress
Priority: P1

Problem/outcome:
Invalid, anomalous, or incompatible data must be detected and assigned an
explicit disposition without quality evaluation mutating the input.

Scope:
- implement independent versioned quality rules and engine capability port;
- evaluate schema, type, range, format, uniqueness, referential, missing/stale,
  duplicate, and anomaly results as applicable to declared capabilities;
- implement pass, warning, reject, quarantine, and hold/explicit-cleaning-required outcomes;
- correlate each result to input, rule version, metadata, run, and stage;
- quarantine or mark according to destination contract and run policy.

Out of scope:
- applying corrections, silent cleaning, false-positive remediation, or a selected statistical platform.

Dependencies:
- BL-004.
- BL-014.
- BL-015.

Architecture traceability:
- `docs/architecture/system-architecture.md` — Quality and Cleaning.
- `docs/architecture/software-architecture.md` — Execution Model and Quality Engine port.
- `docs/software/technology-and-delivery.md` — Quality engine test approach.

Acceptance outcomes:
- [ ] A failed quality rule changes disposition evidence but never mutates the evaluated input.
- [ ] Pass, warning, reject, quarantine, and explicit-cleaning-required outcomes
  are observable and deterministic.
- [ ] Disposition commit and checkpoint behavior remains recoverable and auditable.
- [ ] Cleaning cannot be invoked implicitly by the quality stage.

Required evidence:
- Tests: pass/fail, false-positive/negative fixtures, unknown rule, quarantine, reject, duplicate, cancellation, and recovery.
- Documentation: rule and disposition guide, limitations, quarantine access, and synthetic examples.
- Installation/operations: quarantine retention/access, alerts, and recovery/re-drive procedure.
- Environments: local/test and acceptance policy scenarios.

Open decisions or assumptions:
- **Open:** first quality-rule catalog, anomaly algorithms, and retention/access policy for quarantine.
- **Assumption:** no quality implementation implies hidden correction behavior.

### BL-017: Deliver explicit, deterministic cleaning with complete provenance

Type: Feature  
Status: In Progress
Priority: P1

Problem/outcome:
Approved corrections must be distinguishable from original input, deterministic,
idempotent, authorized, and auditable.

Scope:
- implement cleaning-plan, preview, apply, and result contracts separate from
  quality evaluation;
- require failed rule, correction rule/version, secured input reference, output,
  actor, metadata revision, and result evidence;
- enforce authorization, classification, determinism, idempotency, and
  approved-rule provenance;
- publish a separate cleaned representation without overwriting the preserved input.

Out of scope:
- automatic rule inference, implicit cleaning, and replacement of quarantine policy without explicit approval.

Dependencies:
- BL-005.
- BL-010.
- BL-016.

Architecture traceability:
- `docs/architecture/system-architecture.md` — Quality and Cleaning and Architecture Invariant 9.
- `docs/architecture/software-architecture.md` — Cleaning Engine and Data Flow.
- `docs/software/technology-and-delivery.md` — Cleaning engine test approach.

Acceptance outcomes:
- [ ] A failed quality result cannot trigger cleaning without an approved correction rule and publication decision.
- [ ] Preview and apply produce deterministic, idempotent results and preserve the original input reference.
- [ ] Unauthorized or classified-out cleaning fails closed and records a redacted audit event.
- [ ] Cleaning audit explains who/what/rule changed each output and supports reconciliation after interruption.

Required evidence:
- Tests: preview/apply, pass/fail disposition, idempotency, denied access, redaction, interruption, and audit correlation.
- Documentation: approved-cleaning workflow, provenance, rollback/forward correction, and limitations.
- Installation/operations: actor permissions, cleaning retention, audit review, and restricted recovery.
- Environments: local/test; acceptance sign-off with controlled data; production only by approved policy.

Open decisions or assumptions:
- **Open:** approval actor/mechanism, correction-rule catalog, and retention policy.
- **Assumption:** cleaning is disabled unless explicitly selected by policy.

### BL-018: Deliver deterministic replay and recovery guarantees

Type: Feature  
Status: In Progress
Priority: P1

Problem/outcome:
Operators need replay that is provably equivalent where possible and clearly
unavailable where pinned source position, contract, or rule inputs cannot be
reproduced.

Scope:
- implement replay policy, replay identity, cause, pinned metadata and plan
  inputs, adapter/rule versions, source scope/position, and secret-reference retention;
- compare prior and replay checkpoint/commit evidence to prevent duplicate output;
- refuse replay when a required snapshot, position, adapter behavior, or
  capability/version is unavailable;
- expose recovery and reconciliation commands and evidence without erasing
  original attempts.

Out of scope:
- pretending that unstable sources or unsupported destinations have deterministic replay.

Dependencies:
- BL-011.
- BL-012.
- BL-014.
- BL-016.
- BL-017.

Architecture traceability:
- `docs/architecture/system-architecture.md` — Run Lifecycle, Idempotency and Recovery, Deterministic Replay.
- `docs/architecture/software-architecture.md` — Replay use case and Recovery tests.
- `docs/software/technology-and-delivery.md` — crash and changed-capability scenarios.

Acceptance outcomes:
- [ ] Supported replay pins all required inputs and records original/replayed identities and reason.
- [ ] Missing or changed source position, adapter, rule, or metadata version
  causes explicit unavailability rather than altered guarantees.
- [ ] Replay after any covered durable boundary deduplicates or reconciles committed work.
- [ ] Original run, attempt, checkpoint, failure, and audit evidence remains immutable.

Required evidence:
- Tests: replay availability matrix, changed versions, missing position, stale checkpoint, duplicate delivery, ambiguous commit, and result comparison.
- Documentation: replay guarantees, limitations, operator procedure, and evidence interpretation.
- Installation/operations: restricted replay permissions, backup, stop/resume, and recovery checks.
- Environments: local/test faults and acceptance replay sign-off; production restricted smoke.

Open decisions or assumptions:
- **Open:** replay retention, source snapshot/position guarantees, and operator approval mechanism.
- **Assumption:** replay is opt-in and its guarantee is explicitly stated per run.

### BL-019: Deliver CLI, API, worker, and composition entrypoints

Type: Feature  
Status: In Progress
Priority: P1

Problem/outcome:
Users and operators need thin, authorized entrypoints for metadata control,
operational queries, trigger delivery, and run-worker ownership without moving
business transitions out of application services.

Scope:
- implement approved CLI/local commands, synchronous OpenAPI 3.1 API, scheduler
  worker, run worker, and adapter contract-test host;
- implement protocol/authentication/authorization translation, validation,
  correlation, stable errors, deadlines, and output/exit-code contracts;
- provide one composition boundary for adapter registration, capability startup
  validation, environment configuration, and dependency injection;
- enforce graceful shutdown, readiness, contract/build identity, and worker
  fencing; generate and validate API artifacts.

Out of scope:
- entrypoints containing planning, activation, stage, or state-transition business rules.

Dependencies:
- BL-002.
- BL-005.
- BL-007.
- BL-011.
- BL-012.
- BL-014.
- BL-018.

Architecture traceability:
- `docs/architecture/software-architecture.md` — Entry Points and Composition.
- `docs/software/technology-and-delivery.md` — API, CLI, Worker, and OpenAPI 3.1.
- `docs/implementation/change-workflow.md` — Public API or CLI tests.

Acceptance outcomes:
- [ ] Each CLI/API/worker request calls one use case and cannot bypass validation,
  authorization, planning, or state-transition boundaries.
- [ ] Invalid input, denial, unsupported version, timeout, and internal failure
  have stable, testable outputs and no secret leakage.
- [ ] API/CLI/generated contract drift, startup capability mismatch, and worker
  shutdown/fencing faults fail explicitly.
- [ ] Composition reports build, artifact, and supported contract identity.

Required evidence:
- Tests: handler/command/exit code, OpenAPI, serialization, authorization, version, timeout, shutdown, fencing, and generated freshness.
- Documentation: command/API/worker reference, permissions, examples, errors, and limits.
- Installation/operations: one-/multi-binary install, configuration, health/readiness, startup, and shutdown.
- Environments: local, isolated test, acceptance API/load, production restricted endpoint/worker smoke.

Open decisions or assumptions:
- **Open:** API/worker binary shape, authorization protocol/identity provider,
  external request model, and exact command schema.
- **Assumption:** OpenAPI 3.1/HTTP JSON is used at the synchronous boundary if the proposed technology baseline is approved.

### BL-020: Establish immutable artifact packaging, promotion, and rollback

Type: Technical enabler  
Status: Blocked
Priority: P0

Problem/outcome:
Mimir must promote one identifiable candidate artifact through test, acceptance,
and production without rebuilding it or losing compatibility and rollback evidence.

Scope:
- package approved release archives and OCI images with semantic source/build
  identity, contract versions, provenance, checksums, and integrity metadata;
- automate test deployment, acceptance promotion, approval gates, production
  promotion, smoke, observation, and rollback to a prior accepted artifact;
- validate environment configuration and secret references without resolving secrets into output;
- enforce migration/compatibility, artifact digest, critical-finding, capacity,
  health/readiness, worker fencing, and authorization gates;
- record actor/automation, configuration, metadata revision, timing, and outcome.

Out of scope:
- choosing a registry, signer, hosted environment, or deployment product before ADR approval;
- agent-initiated production deployment.

Dependencies:
- BL-001.
- BL-002.
- BL-004.
- BL-019.

Architecture traceability:
- `docs/software/technology-and-delivery.md` — Environments, CI Pipeline, CD and Promotion, Deployment Compatibility, Release Policy.
- `docs/implementation/change-workflow.md` — Verification and CI/CD, Review and Acceptance.

Acceptance outcomes:
- [ ] The exact accepted digest used in test is promoted to acceptance and
  production without rebuild.
- [ ] Failed migration, health, compatibility, security, audit, or smoke gate
  triggers rollback or an approved forward-fix record.
- [ ] Required checks distinguish passed, failed, not-applicable with scope, and blocked.
- [ ] Release/install evidence identifies supported contracts, configuration revision, migration, and known limits.

Required evidence:
- Tests: artifact identity, provenance, configuration, migration, rollback, health, smoke, and workflow validation.
- Documentation: install/upgrade/backup/uninstall, promotion, rollback, smoke, release notes, and incident recovery.
- Installation/operations: clean supported-target install, acceptance rollback drill, and production runbook.
- Environments: isolated test, controlled acceptance, human-gated production, exact-artifact smoke.

Open decisions or assumptions:
- **Open:** container registry, signing/provenance platform, deployment and
  rollback mechanisms, acceptance data provisioning, and production-smoke permission.
- **Assumption:** OCI and release archives remain packaging formats if the proposed technology baseline is approved.

### BL-021: Publish complete functional, installation, operations, and acceptance documentation

Type: Documentation  
Status: In Progress
Priority: P1

Problem/outcome:
Architecture-level intent is not sufficient for users and operators; each
implemented capability must have executable, environment-specific guidance that
distinguishes current, optional-adapter, and future behavior.

Scope:
- publish functional guides for metadata, discovery, scheduling, planning,
  ingestion, transformation, quality, quarantine, cleaning, replay, API, and CLI;
- publish clean install, upgrade, migration, rollback, uninstall, secret
  reference, permissions, backup, retention, health, and troubleshooting
  procedures for each supported target;
- maintain synthetic executable examples, Mermaid/link/schema/contract checks,
  operations runbooks, known limitations, and release notes;
- define acceptance scenarios tying artifacts, evidence, and sign-off across environments.

Out of scope:
- documenting unimplemented behavior as available, exposing secrets/private endpoints, or replacing feature-specific documentation within each change.

Dependencies:
- BL-014.
- BL-016.
- BL-017.
- BL-018.
- BL-019.
- BL-020.

Architecture traceability:
- `docs/implementation/change-workflow.md` — Documentation, Review and Acceptance, Definition of Done.
- `docs/software/technology-and-delivery.md` — Local Development, Environments, CD and Promotion.
- `README.md` — intended product/toolset and current architecture-definition limitation.

Acceptance outcomes:
- [ ] Every user/operator-visible implemented capability has prerequisites,
  constraints, outputs, failure/recovery, audit, compatibility, and synthetic examples.
- [ ] Install, upgrade, migration, rollback, uninstall, and health commands are
  executed from a clean supported environment before release.
- [ ] Documentation clearly labels current behavior, intended behavior, adapter
  optionality, proposals, and unresolved decisions.
- [ ] Links, Mermaid, schemas, generated API docs, commands, and examples pass documentation validation.

Required evidence:
- Tests: docs/schema/link/generation checks and executable example/acceptance tests.
- Documentation: the deliverable itself, plus README navigation updates when capability or getting-started surfaces change.
- Installation/operations: clean-environment verification for every supported target.
- Environments: local instructions tested; test/acceptance/production procedures tied to immutable artifacts.

Open decisions or assumptions:
- **Open:** supported targets, final commands, production permissions, and deployment-specific runbooks.
- **Assumption:** detailed guides may be organized under `docs/` after delivery conventions are established, but this backlog remains canonical at `docs/backlog/backlog.md`.

### BL-022: Add independently deployable and platform-specific adapter capabilities

Type: Feature  
Status: Blocked
Priority: P2

Problem/outcome:
After core contracts and recovery are proven, users may need selected external
scheduling, discovery, persistence, observability, source, destination, or engine
adapters and independently deployed entrypoints without changing core behavior.

Scope:
- prioritize approved platform/product integrations using the reference port
  and capability contracts;
- add external scheduler/orchestrator, HTTP/JSON adapter-host, persistence,
  telemetry, source, destination, or engine adapters only after focused ADRs;
- preserve declared guarantees, cancellation/timeouts, compatibility, redaction,
  reconciliation, audit, and contract suites per adapter;
- provide registration and process-boundary version negotiation, bounded payloads,
  graceful shutdown, independent artifact identity, and rollback as applicable.

Out of scope:
- moving business transitions into adapters, introducing an event bus/RPC mesh
  without a decision, or making any platform a production prerequisite without approval.

Dependencies:
- BL-001.
- BL-004.
- BL-005.
- BL-010.
- BL-019.
- BL-020.

Architecture traceability:
- `docs/architecture/system-architecture.md` — Adapter Ports and Initial Delivery Shape item 8.
- `docs/architecture/software-architecture.md` — Initial Implementation Increment 8.
- `docs/software/technology-and-delivery.md` — Across Processes and Open Decisions.

Acceptance outcomes:
- [ ] Each integration has an accepted need/ADR, capability matrix, and shared
  plus backend-specific tests.
- [ ] No adapter or cross-process protocol changes domain/application dependency direction.
- [ ] Loss, timeout, unsupported capability, duplicate delivery, changed
  versions, secret exposure, and ambiguous commit fail predictably.
- [ ] Separate deployment preserves the same versioned metadata/run contracts and immutable artifact promotion model.

Required evidence:
- Tests: port conformance, compatibility, external protocol, integration, fault, security, migration, and rollback tests.
- Documentation: capability, install, operations, limits, permission, cost/retention assumptions, and troubleshooting.
- Installation/operations: isolated test resources, health, capacity, shutdown, upgrade, and rollback.
- Environments: local double first; isolated test; acceptance-like selected adapter; human-gated production.

Open decisions or assumptions:
- **Open:** every concrete platform, product, cross-process envelope, and
  independently deployed capability requires its own evidence-backed ADR.
- **Assumption:** P2 does not block the reference implementation's P0/P1 path.

## Coverage verification

| Architecture responsibility | Backlog coverage |
| --- | --- |
| Domain contracts, opaque identities, stable errors | BL-003 |
| Metadata authoring, discovery, validation, activation, resolution | BL-006-BL-008 |
| Versioning, schema drift, compatibility and migration | BL-001, BL-003, BL-006, BL-008, BL-020 |
| Capability declarations and all core ports | BL-003, BL-005, BL-009-BL-013, BL-022 |
| Scheduling and run planning | BL-011 |
| Run, attempt, ownership, state, checkpoint, retry and cancellation | BL-012 |
| Source/destination adapters and safe publication | BL-009, BL-013-BL-014 |
| Normalization/transformation | BL-015 |
| Quality, quarantine, rejection, and explicit cleaning | BL-016-BL-017 |
| Replay and recovery | BL-018, BL-004, BL-012-BL-014 |
| Persistence, audit, lineage, telemetry, and secrets | BL-005, BL-010 |
| Authorization, classification, redaction, and security review gates | BL-005, all affected item evidence |
| Contract, dependency, race, fault, concurrency, and migration tests | BL-002, BL-004, item-specific acceptance evidence |
| CLI, API, workers, composition, adapter test host | BL-019 |
| Local development, CI/CD, promotion, rollback, acceptance and smoke | BL-002, BL-020 |
| Functional, installation, operations, release, and acceptance docs | BL-021 plus same-change documentation in every item |
| Additional/external replaceable infrastructure and independent deployment | BL-022 |

## Dependency validation

The dependency graph is acyclic by construction and follows ascending
identifiers. BL-001 and BL-009 are decision gates; no item may be promoted to
Ready until its referenced open decisions are resolved. The backlog contains no
same-level forward dependency: every referenced prerequisite has an earlier
identifier and definition.

Before implementation, each selected item must be decomposed in a separate
`BL-<id>-<short-name>.md` refinement with observable Given/When/Then criteria,
a first failing test, compatibility/migration/rollback detail, and all four
environment-verification levels. No current item is refined or ready merely
because this roadmap exists.
