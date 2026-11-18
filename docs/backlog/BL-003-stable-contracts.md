# BL-003 refinement — Deliver stable domain, capability, error, and port contracts

Parent: [BL-003](./backlog.md#bl-003-deliver-stable-domain-capability-error-and-port-contracts)  
Workflow: [`docs/implementation/change-workflow.md`](../implementation/change-workflow.md)

## Outcome and current evidence

**Current:** There are no source contracts, generated schemas, or tests. **Required:** domain/application layers depend on ports, capabilities are explicit and versioned, and adapter failures map to stable Mimir categories. **Open:** BL-001.2 must settle exact compatibility, canonical record, and error-envelope details.

The two children separate core values/errors from substitutable port/capability interfaces; together they cover BL-003 without implementing adapters.

Owner / target milestone for all children: **Open** — no owner or milestone is recorded in the repository; assign before moving any child to Ready.

Blocker tracking: For each **Blocked** child, its unresolved dependencies below are the blocker; the next action is to complete the first unresolved prerequisite in that list. Blocker owner is **Open** and must be assigned by maintainers.

## Child work items

### BL-003.1: Define immutable domain values and stable failure categories

Type: Technical enabler  
Status: Blocked  
Priority: P0

Problem: Modules cannot share invariants or report failures consistently without technology-independent domain contracts.

Scope:
- Implement opaque IDs/value objects and constructor-enforced invariants for metadata revision, effective policy, plan, run/attempt, checkpoint, records/batches, lineage, rules, adapter versions, and secret references.
- Define stable failure categories and fields for stage, scope, retryability, cause reference, commit certainty, and reconciliation action.

Out of scope:
- Port implementation, persistence, wire serialization, and concrete record-transfer format.

Affected modules and contracts:
- `contracts/domain`, `contracts/errors`; immutable values consumed by metadata, planning, state, and execution.

Acceptance criteria:
- [ ] Given valid, invalid, and boundary values, when domain constructors run, then valid instances satisfy invariants and invalid instances return stable categorized errors.
- [ ] Given a failure with unknown commit outcome, when represented, then its category and commit certainty require reconciliation and cannot be interpreted as safe retry.
- [ ] Given a domain package build, when dependency checks run, then it imports no adapter, entrypoint, or infrastructure package.

TDD first:
- First failing test: table-driven invariant test for a checkpoint without durable commit evidence; it must reject before any state can be constructed.
- Additional coverage: identifier boundaries, immutable value behavior, error mapping, serialization/compatibility fixtures after BL-001.2.

Implementation notes:
- Preserve source positions and provenance by reference; do not settle unapproved external data representation.

Documentation updates:
- Functional: error categories and contract semantics for future callers.
- Installation: no new runtime installation.
- Operations/release: document stable error and version reporting expectations.
- Architecture/ADR: link to approved BL-001.2 contract policy.

Security and data impact:
- Secret references are opaque; domain errors must not carry resolved secrets or raw sensitive payloads.

Compatibility, migration, and rollback:
- Contract changes require previous-version fixtures and unsupported-version rejection per BL-001.2; no persistence selected here.

Environment verification:
- Local: unit and dependency tests are hermetic.
- Test: pinned CI runs domain and compatibility suite.
- Acceptance: contract compatibility suite is run against candidate versions before promotion.
- Production promotion or smoke: N/A for a domain-only change; expose contract version for later restricted smoke.

Dependencies:
- BL-001.2.
- BL-002.1.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus the invariant/error evidence above.

### BL-003.2: Define versioned capability and port contracts

Type: Technical enabler  
Status: Blocked  
Priority: P0

Problem: Consumers need replaceable ports that state guarantees and explicitly reject unsupported operations.

Scope:
- Define versioned contracts for metadata repository/discovery, source, destination, scheduler, run-state, secret resolver, transformation, quality, cleaning, audit, and telemetry.
- Define capability declarations for operations, versions, scope/position, guarantees, limits, concurrency, security/classification, and unsupported operations.
- Specify dependency-direction checks and stable adapter failure mapping.

Out of scope:
- Concrete adapters, external wire protocol, or a chosen storage/transport product.

Affected modules and contracts:
- Application/domain-owned ports; adapter capability and error boundaries.

Acceptance criteria:
- [ ] Given a port implementation omitting a required operation, when the planner/test contract evaluates it, then the omission is explicit and unsafe use is rejected before side effects.
- [ ] Given adapter-specific exceptions, when they cross a port boundary, then consumers observe only stable Mimir failures and cause references.
- [ ] Given a module import graph, when architecture checks run, then inward modules importing adapters fail the check.

TDD first:
- First failing test: a fake adapter advertising no required capability is rejected with a stable unsupported-capability error and zero calls to its execution methods.
- Additional coverage: shared port contract cases for cancellation, timeout, limits, version compatibility, and translated errors (expanded in BL-004).

Implementation notes:
- Exact signatures/serialization remain blocked until BL-001.2; build contract tests after approved shapes exist.

Documentation updates:
- Functional: port and capability reference with explicit guarantees.
- Installation: adapter implementer prerequisites and supported contract versions.
- Operations/release: startup/planner behavior for missing or changed capability.
- Architecture/ADR: maintain module-boundary and version references.

Security and data impact:
- Include classification, least-privilege, secret scoping, and redaction capabilities; no raw secret in contracts.

Compatibility, migration, and rollback:
- Test previous supported capability/port versions and reject unsupported versions without side effects; no data migration here.

Environment verification:
- Local: hermetic contract and architecture tests.
- Test: all shared fixtures run in CI.
- Acceptance: adapters used by acceptance must pass the same contract suite.
- Production promotion or smoke: N/A for contract definition; require version/health compatibility before later promotion.

Dependencies:
- BL-003.1.
- BL-004.1.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus contract and dependency evidence above.

## Parent coverage and blockers

Children cover domain identifiers/invariants/errors, all named port families, versioned capability declarations, stable error translation, and dependency direction. Both are blocked on the BL-001.2 decisions; .2 also requires the initial contract harness in BL-004.1.
