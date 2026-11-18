# BL-007 refinement — Deliver metadata authoring, validation, activation, and policy resolution

Parent: [BL-007](./backlog.md#bl-007-deliver-metadata-authoring-validation-activation-and-policy-resolution)  
Workflow: [`docs/implementation/change-workflow.md`](../implementation/change-workflow.md)

## Outcome and current evidence

**Current:** No metadata use cases, repository port implementation, audit path, or tests exist. **Required:** drafts have no execution authority; only validated and authorized revisions can be activated; active revisions are immutable and policy resolution is deterministic. **Open:** approval and durable audit decisions remain BL-001.3 and BL-009.

The three children cover repository-backed candidate history, validation-gated conditional activation/audit, and deterministic policy resolution. Together they satisfy BL-007 while leaving discovery to BL-008 and concrete persistence selection to BL-009/010.

Owner / target milestone for all children: **Open** — no owner or milestone is recorded in the repository; assign before moving any child to Ready.

Blocker tracking: For each **Blocked** child, its unresolved dependencies below are the blocker; the next action is to complete the first unresolved prerequisite in that list. Blocker owner is **Open** and must be assigned by maintainers.

## Child work items

### BL-007.1: Persist candidate metadata revisions and validation reports through a repository port

Type: Feature  
Status: Blocked  
Priority: P0

Problem: Authors need an append-only, inspectable draft/revision workflow that cannot itself authorize execution.

Scope:
- Implement authoring, revision identity, candidate append/load/list, immutable validation reports and audit intent using the repository port.
- Provide a deterministic local/test double; keep production persistence replaceable.

Out of scope:
- Activation, discovery, policy resolution, and a production database choice.

Affected modules and contracts:
- Metadata authoring application service, metadata revision, repository and audit-intent ports.

Acceptance criteria:
- [ ] Given authorized candidate metadata, when authored, then a new immutable draft revision and author audit intent are returned.
- [ ] Given a draft or failed validation report, when queried for run eligibility, then it is never returned as active/executable.
- [ ] Given a repository failure or duplicate revision identity, when authoring occurs, then no partially successful revision is reported.

TDD first:
- First failing test: application test authors an invalid candidate and asserts it is persisted only as non-executable draft with immutable validation evidence.
- Additional coverage: repository contract, duplicate identity, authorization denial, audit failure behavior and previous metadata version fixtures.

Implementation notes:
- Durable implementation follows BL-009; do not imply an in-memory test double meets production durability requirements.

Documentation updates:
- Functional: authoring lifecycle, draft status, validation report and recovery.
- Installation: local/test repository configuration only after BL-009 profile.
- Operations/release: backup/restore and retention requirements; implementation-specific details follow BL-010.
- Architecture/ADR: document audit intent and repository port use.

Security and data impact:
- Enforce BL-005 authorization; secret references only, classify metadata and redact reports.

Compatibility, migration, and rollback:
- Append-only revisions; compatibility fixtures and no destructive schema migration. Rollback retains previous revision and records supersession.

Environment verification:
- Local: use deterministic test repository, no production credentials.
- Test: repository contract and use-case tests in CI.
- Acceptance: non-production durable repository/recovery checks after BL-009/010.
- Production promotion or smoke: not enabled until adapter and policy approval; later verify read-only metadata and audit correlation.

Dependencies:
- BL-004.1.
- BL-005.2.
- BL-006.2.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus repository/use-case evidence above.

### BL-007.2: Validate and conditionally activate immutable metadata revisions

Type: Feature  
Status: Blocked  
Priority: P0

Problem: A candidate must not authorize execution unless validation and required approval succeed, including during concurrent activation.

Scope:
- Implement validation use case integration, authorized activation, active-revision promotion, supersession, conditional concurrency and durable audit intent.
- Reject stale, invalid, unauthorized, unsupported, or incompatible revisions before activation side effects.

Out of scope:
- Discovery-generated proposals, external approval UI, or a selected persistence backend.

Affected modules and contracts:
- Metadata validation/activation application service, repository conditional activation, authorization and audit boundaries.

Acceptance criteria:
- [ ] Given an invalid, stale, unauthorized, or unsupported revision, when activation is requested, then active revision remains unchanged and a redacted failure is recorded.
- [ ] Given two concurrent incompatible activations, when both race, then at most one becomes active and the other receives a stable conflict.
- [ ] Given successful activation, when later metadata changes, then the activated revision and runs pinned to it remain unchanged.

TDD first:
- First failing test: race two activation requests against a controlled repository double and assert exactly one conditional activation succeeds.
- Additional coverage: validation gates, authorization denial, audit interruption/recovery, prior-version compatibility and no-partial-side-effect cases.

Implementation notes:
- Approval semantics come from BL-001.3; durable atomicity/recovery must follow BL-009 and BL-010.

Documentation updates:
- Functional: validation, approval, activation, supersession and failure behavior.
- Installation: required authorization and repository config.
- Operations/release: concurrent activation, audit recovery, backup and rollback.
- Architecture/ADR: update activation policy if approval choice changes the accepted boundary.

Security and data impact:
- Security-sensitive: enforce authorization, record redacted decision audit, protect classified metadata and secrets.

Compatibility, migration, and rollback:
- Active revision is immutable; rollback means an explicit superseding activation, never rewriting history. Require compatible readers and tested recovery.

Environment verification:
- Local: authorization and activation race tests with synthetic identities.
- Test: CI repository contract, version and race suites.
- Acceptance: activate/rollback a synthetic revision with isolated non-production policy.
- Production promotion or smoke: no production activation until approval; later verify read-only active revision and audit event.

Dependencies:
- BL-007.1.
- BL-006.2.
- BL-005.2.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, including focused security review.

### BL-007.3: Resolve effective policy deterministically with override provenance

Type: Feature  
Status: Blocked  
Priority: P0

Problem: Planning needs effective source/object/column/rule settings with deterministic precedence and traceable origins.

Scope:
- Resolve source defaults, object overrides, column overrides, and rule-level policy for an active immutable revision.
- Preserve the origin of each resolved value and validate unresolved/ambiguous conflicts rather than silently choosing.

Out of scope:
- Scheduling, adapter matching, schema discovery and executing policy actions.

Affected modules and contracts:
- Effective Policy domain service, active metadata lookup and immutable plan inputs.

Acceptance criteria:
- [ ] Given conflicting source/object/column values, when resolution runs, then the documented precedence is deterministic and each effective value names its origin.
- [ ] Given an invalid or ambiguous override, when resolution runs, then it returns a stable scoped validation error and no partial policy.
- [ ] Given a draft or superseded revision, when resolved for a new plan, then it is rejected unless policy explicitly authorizes that revision.

TDD first:
- First failing test: table-driven inheritance test with all three override levels asserts exact effective values and origin paths.
- Additional coverage: missing values, rule-level policy, invalid precedence, classification, immutable snapshot and version compatibility.

Implementation notes:
- Record effective policy as immutable run input; never mutate metadata or infer transformation/cleaning.

Documentation updates:
- Functional: precedence rules and examples using synthetic metadata.
- Installation: no additional install; describe validation tool only.
- Operations/release: explain policy-origin diagnostics and active revision pinning.
- Architecture/ADR: keep precedence consistent with approved metadata model.

Security and data impact:
- Apply classification and authorization constraints; omit secret values from resolved policy and diagnostics.

Compatibility, migration, and rollback:
- Version resolution semantics if precedence changes; test old policy snapshots and document migration/rollback.

Environment verification:
- Local: hermetic table-driven domain tests.
- Test: schema and compatibility suite in CI.
- Acceptance: compare resolved policy for a representative synthetic revision.
- Production promotion or smoke: N/A for pure resolution; future plan preflight must expose revision identity without sensitive values.

Dependencies:
- BL-007.2.
- BL-006.2.
- BL-001.3.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus deterministic resolution evidence above.

## Parent coverage and blockers

Children cover authoring, append-only candidate/revision history, immutable validation reports, authorized conditional activation and supersession, durable audit intent, and inheritance resolution with origin. Discovery is explicitly excluded by BL-007 and belongs to BL-008. Work remains blocked by governance and reference repository/audit decisions.
