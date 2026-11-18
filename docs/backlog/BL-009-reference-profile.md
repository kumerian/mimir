# BL-009 refinement — Approve first reference-adapter and persistence profile

Parent: [BL-009](./backlog.md#bl-009-approve-first-reference-adapter-and-persistence-profile)  
Workflow: [`docs/implementation/change-workflow.md`](../implementation/change-workflow.md)

## Outcome and current evidence

**Current:** ADR-0002 selects a standard-library local JSONL source, append-only destination, separate metadata/state/data/audit files, owner-only permissions, process lock, tail recovery, deduplicated commit, and checkpoint-after-commit. These decisions are accepted for implementation under user delegation, not production approval.

These are decision items, not implementation work. The three ADRs separate persistence/audit profile, recovery/delivery semantics, and reference source/destination capability profile.

Owner / target milestone for all children: **Open** — no owner or milestone is recorded in the repository; assign before moving any child to Ready.

Blocker tracking: For each **Blocked** child, its unresolved dependencies below are the blocker; the next action is to complete the first unresolved prerequisite in that list. Blocker owner is **Open** and must be assigned by maintainers.

## Child work items

### BL-009.1: Decide the replaceable metadata, state, and audit persistence profile

Type: Architecture decision  
Status: In Progress
Priority: P0

Problem: Metadata/state/audit must be durable and separable, but no backing product or local-reference persistence mechanism is approved.

Scope:
- Decide the minimum development/reference metadata repository, run-state store, and durable audit/outbox or retryable journal semantics.
- Decide whether a development-only secret-reference resolver is part of the local profile; it must not imply production suitability.
- Specify ownership, data separation, backup/restore, retention, failure, migration and alternatives; name production suitability limits.

Out of scope:
- Selecting a production database/cloud or implementing persistence adapters.

Affected modules and contracts:
- Metadata repository, run-state, audit store, event recovery, local development profile.

Acceptance criteria:
- [x] Given the required metadata/state/audit outcomes, when alternatives are evaluated, then an ADR identifies the selected capability profile or explicitly defers it with owner and dependent items. ADR-0002 selects local files and expressly defers production products.
- [x] Given required audit-sink unavailability, when the profile is reviewed, then it defines how the outcome remains recoverable without claiming unsupported atomicity. Required audit append is distinct from best-effort telemetry and is retried/reconciled; no cross-file atomicity is claimed.
- [x] Given local and production constraints, when the ADR is approved, then local-reference choices are explicitly not presumed production-ready. ADR-0002 states single-process, local-disk-only limitations; independent maintainer review remains a promotion gate.

TDD first:
- First failing test: documentation validation rejects an ADR without separate data ownership, audit failure behavior, backup/restore, retention, migration, rollback, alternatives and status. Code tests are N/A because this is an architecture decision.
- Additional coverage: ADR schema, links to BL-010/011/012/014 and architecture terms.

Implementation notes:
- Avoid naming products until the owner records an approved decision. Describe capability requirements independent of brand.

Documentation updates:
- Functional: N/A; record observable durability guarantees for later docs.
- Installation: define local resource setup/cleanup and limitations after decision.
- Operations/release: backup/restore, retention, journal recovery, health and migration constraints.
- Architecture/ADR: publish decision and update open choices.

Security and data impact:
- Decision must cover encryption/access controls, retention/deletion, audit integrity, and classification; no credentials or private endpoints in docs.

Compatibility, migration, and rollback:
- Define schema/state compatibility, migration sequence, failure handling and rollback limits; destructive migration requires separate approval.

Environment verification:
- Local: ADR and links validate; local profile requirements are explicit.
- Test: specify contract and failure tests required of implementation.
- Acceptance: state durability/recovery proof against the exact candidate.
- Production promotion or smoke: record production suitability as open or constrained; no product is promoted by this decision alone.

Dependencies:
- BL-001.1.
- BL-001.2.
- BL-001.3.
- BL-003.2.
- BL-007.1.
- BL-008.1.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full; human architecture approval and ADR evidence are mandatory.

### BL-009.2: Decide checkpoint, concurrency, commit, and replay guarantees

Type: Architecture decision  
Status: In Progress
Priority: P0

Problem: Runtime cannot implement safe recovery until durable boundaries, fencing, checkpoint granularity, and replay limits are specified.

Scope:
- Decide supported checkpoint granularity, ownership/fencing primitive requirements, commit/reconciliation strategy classes, event/request identity, retry semantics, and replay guarantees.
- Specify crash boundary behavior and explicitly reject combinations that cannot meet delivery guarantees.

Out of scope:
- Claiming exactly-once behavior from a state store alone or implementing a chosen backend.

Affected modules and contracts:
- Run/attempt transitions, source acknowledgement, destination commit, checkpoint, replay and run request identity.

Acceptance criteria:
- [x] Given each possible crash boundary from source read through acknowledgement, when the decision is reviewed, then committed/unknown/uncommitted outcomes and recovery action are stated. ADR-0002 includes the local boundary matrix and notes acknowledgement is unsupported.
- [x] Given missing source position or destination reconciliation, when replay is assessed, then deterministic replay is declared unavailable or constrained rather than guaranteed. Replay is bounded to unchanged snapshots and reconciled destination identity.
- [x] Given a concurrent stale worker, when ownership policy is applied, then the required conditional transition/fencing behavior is explicit. Local conditional journal appends and attempt owner tokens do not claim distributed fencing.

TDD first:
- First failing test: documentation scenario validator rejects a decision that leaves any durable boundary without an outcome/recovery rule. Code tests are N/A because this is a decision.
- Additional coverage: compatibility and ADR links to BL-004.2/BL-012/BL-013.

Implementation notes:
- Select guarantees and capability requirements, not a scheduler, broker, database, or deployment product.

Documentation updates:
- Functional: record supported retry/replay claims after decision.
- Installation: state local profile requirements only after approval.
- Operations/release: runbook requirements for ambiguous commits, stale workers and resume.
- Architecture/ADR: specify consequences and alternatives, preserving invariant 6/7.

Security and data impact:
- Include authorization around replay, secret references without values, and audit requirements for ambiguous outcomes.

Compatibility, migration, and rollback:
- Define versioning for state/checkpoint and safe expand/contract or explicit downtime requirements; no state migration implemented.

Environment verification:
- Local: ADR completeness and links validate.
- Test: enumerate fault-injection cases required before implementation is accepted.
- Acceptance: specify restart/replay and stale-owner scenarios against candidate.
- Production promotion or smoke: no runtime change; define restricted health/fencing checks and rollback evidence.

Dependencies:
- BL-001.2.
- BL-003.1.
- BL-004.2.
- BL-007.1.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full; human architecture approval and ADR evidence are mandatory.

### BL-009.3: Decide minimum reference source and destination capabilities

Type: Architecture decision  
Status: In Progress
Priority: P0

Problem: BL-013 cannot implement a meaningful reference path without an approved capability set and explicit delivery limits.

Scope:
- Choose or defer the minimum bounded-batch source/destination profile and define scope, position, stable identity, acknowledgement, idempotency, durable commit evidence, reconciliation, schema observation, security and limits.
- Specify alternatives and why the profile is replaceable through approved BL-003 ports.

Out of scope:
- Any unapproved platform product or assuming that a local reference is suitable for production.

Affected modules and contracts:
- Source/destination adapters, run planner, commit/checkpoint protocol, local setup and test dependencies.

Acceptance criteria:
- [x] Given a reference profile, when capabilities are documented, then every supported and unsupported operation and its guarantee/limit is explicit. File, state, audit, telemetry, discovery, and secret adapters declare versioned capabilities and exclusions.
- [x] Given a repeated batch and an ambiguous commit, when the profile is evaluated, then the required deduplication/reconciliation result is unambiguous. Stable batch identity is reconciled before retry.
- [x] Given the system's platform-neutral contract, when alternatives are assessed, then no provider-specific behavior leaks into core ports.

TDD first:
- First failing test: capability-profile validation rejects a profile that lacks a declared stable identity or commit-reconciliation behavior required by BL-009.2. Code tests are N/A because the profile selection is a decision.
- Additional coverage: ADR completeness, capability matrix consistency, links to BL-013/014 and contract terms.

Implementation notes:
- If maintainers defer a product choice, downstream implementation stays Blocked; do not substitute a guessed in-memory/file/database adapter.

Documentation updates:
- Functional: document reference-path guarantees and exclusions once selected.
- Installation: prerequisites, cleanup, limits, and no-production-credential setup.
- Operations/release: reconciliation, backup, capacity and production suitability limitations.
- Architecture/ADR: record alternatives, consequences, migration and rollback.

Security and data impact:
- Evaluate access scope, classification, secret handling, data retention and synthetic test suitability.

Compatibility, migration, and rollback:
- State source/destination contract versions, data ownership, migration and rollback boundaries; preserve replaceability.

Environment verification:
- Local: profile/ADR validation; no product install until approved.
- Test: specify adapter contract and fault tests required.
- Acceptance: define isolated end-to-end capability evidence.
- Production promotion or smoke: no production use authorized by this decision; record any separate approval gate.

Dependencies:
- BL-009.2.
- BL-003.2.
- BL-004.1.
- BL-008.1.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full; human architecture/product approval and ADR evidence are mandatory.

## Parent coverage and blockers

Children resolve persistence/audit, recovery/checkpoint/concurrency, and source/destination reference-profile decisions. BL-009 itself remains a blocker until these ADRs are approved; the repository contains no evidence that any product or guarantee has been selected.
