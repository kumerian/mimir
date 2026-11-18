# BL-010 refinement — Provide persistence, audit, telemetry, and secret-reference adapters

Parent: [BL-010](./backlog.md#bl-010-provide-persistence-audit-telemetry-and-secret-reference-adapters)  
Workflow: [`docs/implementation/change-workflow.md`](../implementation/change-workflow.md)

## Outcome and current evidence

**Current:** The local profile stores metadata, immutable plans, owner-fenced lifecycle/checkpoint state, destination/quarantine/cleaned data, and append-only audit separately. Required audit failures block success and can be reconciled; allowlisted correlated telemetry is best-effort and local. Metrics/traces exporters and composed secret resolution are absent; local retention is operator-owned with no automatic expiry.

The three children cover the selected metadata/state persistence, durable audit and telemetry correlation, and development secret-resolution adapter. Each remains technology-neutral until profile ADRs are accepted.

Owner / target milestone for all children: **Open** — no owner or milestone is recorded in the repository; assign before moving any child to Ready.

Blocker tracking: For each **Blocked** child, its unresolved dependencies below are the blocker; the next action is to complete the first unresolved prerequisite in that list. Blocker owner is **Open** and must be assigned by maintainers.

## Child work items

### BL-010.1: Implement approved metadata and run-state repository adapters

Type: Technical enabler  
Status: In Progress
Priority: P0

Problem: Metadata revisions, plans, attempts and checkpoints need a durable implementation behind their ports.

Scope:
- Implement only BL-009.1-approved metadata repository and state-store behavior, preserving separate logical ownership and conditional transitions.
- Run shared repository contracts, version/compatibility, failure, cancellation, timeout, backup/restore and migration checks.

Out of scope:
- Source/destination data storage, production backend selection outside approved ADR, or combining state with ingested data authority.

Affected modules and contracts:
- Metadata repository, run-state store, conditional activation/transition, checkpoint persistence and audit intents.

Acceptance criteria:
- [x] Given two competing conditional updates, when repository/state methods execute, then stale writes are rejected and prior evidence remains intact. Repository activation and state-store tests cover stale preconditions, competing initial owners, and retry history.
- [x] Given an unsupported schema/state version, when opened or migrated, then operation fails before partial state change. Metadata and run-state adapters reject unsupported versions; no previous persisted format is claimed.
- [x] Given separate metadata and execution state records, when stored, then each retains its domain authority and version independently. Metadata revisions/pointers and run plans/transitions/checkpoints remain in separate directories and contracts.

TDD first:
- First failing test: shared state-store contract attempts a transition with a stale version and asserts rejection without overwriting newer state.
- Additional coverage: repository contracts, durable reopen, interruption, migration/rollback, backup/restore, concurrent activation and synthetic fixtures.

Implementation notes:
- No implementation is Ready until BL-009.1 selects and approves the reference profile.

Documentation updates:
- Functional: persistence behavior visible to metadata/run use cases.
- Installation: approved local/test dependency setup and clean teardown.
- Operations/release: durability, backup, restore, retention, migration and recovery.
- Architecture/ADR: adapter mapping and data ownership must match BL-009.1.

Security and data impact:
- Apply classification, least-privilege storage access, encryption policy if selected, retention/deletion, and no secret values in stored plans.

Compatibility, migration, and rollback:
- Test previous supported persisted schema and unsupported rejection; use approved expand/contract sequence or approved downtime. Preserve provenance and rollback limits.

Environment verification:
- Local: approved deterministic reference profile and repository contract tests.
- Test: isolated persistence integration and fault tests with synthetic data.
- Acceptance: durability, restore, migration and concurrency against exact candidate.
- Production promotion or smoke: only after separate suitability approval; restricted state-health and recovery checks, not a backend choice by this item.

Dependencies:
- BL-004.1.
- BL-005.1.
- BL-007.1.
- BL-009.1.
- BL-009.2.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus adapter, migration and recovery evidence above.

### BL-010.2: Preserve required audit outcomes and correlate operational telemetry

Type: Technical enabler  
Status: In Progress
Priority: P0

Problem: Activation, access, cleaning, and ambiguous-commit evidence must not disappear with best-effort telemetry.

Scope:
- Implement BL-009.1-approved durable audit journal/outbox/retry semantics and append-only audit port behavior.
- Propagate correlation through revision, plan, run, attempt, stage, operation, adapter/rule and record identity where applicable.
- Emit structured logs/metrics/traces with redaction and explicit distinction from durable audit.

Out of scope:
- Selecting telemetry vendor, losing required audit on export failure, or requiring operational telemetry to be durable if policy says otherwise.

Affected modules and contracts:
- Audit store, telemetry port, correlation context, application outcome boundaries and redaction.

Acceptance criteria:
- [x] Given audit sink or telemetry exporter unavailable, when a required business outcome occurs, then its audit intent remains recoverable and operational export failure does not falsely mark it lost. Fault tests deny the required run audit append and verify retry reconciliation; a separate telemetry sink failure does not change run outcome.
- [x] Given one run, when stage events are emitted, then applicable correlation identities match across audit and telemetry. The end-to-end test checks run, plan, revision, and attempt identity.
- [x] Given a synthetic secret/sensitive value, when events are serialized, then it is absent unless an approved protected policy explicitly permits it. Telemetry accepts an allowlisted envelope and rejects unknown fields, event names, and failure categories; audit rejects unbounded operation/actor text and contains no payload fields.

TDD first:
- First failing test: inject audit sink failure after an activation outcome and assert the journal/outbox retains exactly one recoverable event and activation does not report unrecorded success.
- Additional coverage: duplicate audit delivery, restart/replay, correlation, sampling, exporter failure, redaction, retention and deletion.

Implementation notes:
- Apply durable-vs-best-effort policies from BL-001.3/BL-009.1; do not invent atomicity.

Documentation updates:
- Functional: describe audit and telemetry visibility/limitations.
- Installation: configure approved local journal/exporter and synthetic data.
- Operations/release: audit recovery, retention, sampling, alerts, backup, export failure and runbook.
- Architecture/ADR: preserve separate audit/telemetry loss semantics.

Security and data impact:
- Audit contains access/activation outcomes but not secrets/raw sensitive values; require focused security review.

Compatibility, migration, and rollback:
- Version audit envelope, preserve append-only history, and document journal migration/retention. Roll back exporter separately from required audit path.

Environment verification:
- Local: deterministic in-process sink and failure/restart tests.
- Test: isolated sink integration, redaction and correlation suite.
- Acceptance: durability/export outage and retention scenario against exact candidate.
- Production promotion or smoke: only after policy approval; verify audit continuity and telemetry correlation with restricted synthetic-safe events.

Dependencies:
- BL-004.2.
- BL-005.1.
- BL-007.2.
- BL-009.1.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, including focused security review and audit evidence.

### BL-010.3: Implement approved development secret-reference resolver adapter

Type: Technical enabler  
Status: In Progress
Priority: P0

Problem: Local runs and adapters need a safe resolver implementation without embedding credential values or choosing a production secret store.

Scope:
- Implement the BL-009.1/profile-approved development resolver, bounded resolution, cancellation, timeout, denied references and redacted failure mapping.
- Reuse resolver shared contract and prove resolved values remain inside adapter boundary.

Out of scope:
- Production identity/secret system, ambient developer credentials, plaintext metadata, or logging resolved values.

Affected modules and contracts:
- Secret resolver port, composition registration, local configuration and adapter security tests.

Acceptance criteria:
- [ ] Given a permitted synthetic reference, when an authorized bounded adapter operation runs, then the value is available only within that operation and not in plan/state/output. The local resolver is tested as a synchronous callback port but is not composed into a source/destination adapter or authorization provider. Local file adapters reject unconsumed secret references before activation, discovery, or run rather than silently ignoring them.
- [ ] Given an unknown, denied, expired, timed-out, or cancelled reference, when resolved, then the adapter fails closed with a stable redacted category. Unknown/missing, timed-out, and cancelled references are covered; expiry semantics and hosted authorization are not part of the local environment-secret adapter.

TDD first:
- First failing test: resolver conformance test injects a canary value and asserts it cannot be serialized after the bounded operation completes.
- Additional coverage: permission denial, expiry, timeout, cancellation, redaction, reference-version compatibility and cleanup.

Implementation notes:
- This is a development/reference adapter only; production use requires a distinct approved profile.

Documentation updates:
- Functional: reference and failure semantics.
- Installation: safe local configuration and cleanup using synthetic values.
- Operations/release: rotation/expiry, diagnostics and production non-support statement.
- Architecture/ADR: state selected reference capability and limitations.

Security and data impact:
- High security impact; narrow scope/lifetime, no logs or outputs, deny by default, focused review.

Compatibility, migration, and rollback:
- Version reference formats; never automatically migrate plaintext credentials; rollback removes resolver registration without changing metadata history.

Environment verification:
- Local: synthetic dev resolver and redaction tests.
- Test: isolated contract suite with synthetic references.
- Acceptance: only approved non-production secret references, verified independently.
- Production promotion or smoke: explicitly not production-capable absent separate ADR/approval; no secret values in smoke output.

Dependencies:
- BL-005.1.
- BL-009.1.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, including focused security review.

## Parent coverage and blockers

Children cover repository/state adapters, durable audit and correlation/redaction, and secret resolution through replaceable ports, with integration, compatibility, backup, and recovery evidence. All implementation is blocked on the relevant BL-009 decisions; no persistence or secret product has been selected.
