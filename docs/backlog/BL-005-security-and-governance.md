# BL-005 refinement — Enforce secret-reference, authorization, classification, and redaction policy

Parent: [BL-005](./backlog.md#bl-005-enforce-secret-reference-authorization-classification-and-redaction-policy)  
Workflow: [`docs/implementation/change-workflow.md`](../implementation/change-workflow.md)

## Outcome and current evidence

**Current:** There is no source, identity integration, secret resolver, policy, or runtime telemetry. **Required:** secret values stay out of metadata/plans/logs, authorization is enforced at control/planning/runtime boundaries, and data classification constrains use. **Open:** BL-001.3 must define identity inputs, policy, classes, retention, and approval.

Children cover contract-level secret handling and enforcement/redaction, then classification policy at all decision points. Neither selects an identity provider or secret store.

Owner / target milestone for all children: **Open** — no owner or milestone is recorded in the repository; assign before moving any child to Ready.

Blocker tracking: For each **Blocked** child, its unresolved dependencies below are the blocker; the next action is to complete the first unresolved prerequisite in that list. Blocker owner is **Open** and must be assigned by maintainers.

## Child work items

### BL-005.1: Enforce opaque secret references and bounded resolution

Type: Technical enabler  
Status: Blocked  
Priority: P0

Problem: Credentials must be referenced and resolved just in time without entering core state or outputs.

Scope:
- Implement opaque secret-reference values and resolver port behavior with bounded operation scope, timeout, cancellation, denial and stable failure categories.
- Ensure metadata, run plans, errors, audit, lineage, logs and traces do not contain resolved secret values.

Out of scope:
- Selecting or implementing a production secret store or identity provider.

Affected modules and contracts:
- Secret reference domain value, secret resolver port, adapter gateway and redaction filters.

Acceptance criteria:
- [ ] Given a resolved synthetic secret, when a plan, error, log, trace, lineage, or audit event is serialized, then the secret value is absent.
- [ ] Given a denied, unavailable, timed-out, or cancelled reference, when an adapter requests it, then a stable failure is returned and no unscoped operation proceeds.

TDD first:
- First failing test: inject a canary secret into a resolver result and assert recursive serialized outputs contain the reference but never the canary value.
- Additional coverage: denied access, unsupported reference type, timeout, cancellation, lifetime boundary, adapter error cause redaction.

Implementation notes:
- Keep resolution inside infrastructure boundary and only immediately before an authorized adapter operation.

Documentation updates:
- Functional: reference semantics and failure behavior.
- Installation: configure a non-secret development reference; no plaintext credential examples.
- Operations/release: rotation/lifetime, denied-reference diagnosis, and redaction expectations.
- Architecture/ADR: conform to BL-001.3 and BL-003 port contracts.

Security and data impact:
- High security impact; least privilege, secret lifetime, logs/output scanning, and denied access require focused security review.

Compatibility, migration, and rollback:
- Version secret-reference contract; do not migrate plaintext values automatically. Rollback must not reintroduce secret serialization.

Environment verification:
- Local: synthetic resolver and canary redaction tests.
- Test: isolated resolver contract suite, no ambient credentials.
- Acceptance: independently managed non-production references and denial/redaction checks.
- Production promotion or smoke: no production resolver chosen; later restricted smoke verifies access and redaction without exposing secret.

Dependencies:
- BL-001.3.
- BL-003.1.
- BL-004.1.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, including focused security review.

### BL-005.2: Enforce authorization and classification at control and planning boundaries

Type: Technical enabler  
Status: Blocked  
Priority: P0

Problem: User, source, destination, rule, and data-classification policy must be enforced by use cases rather than only by a UI or adapter.

Scope:
- Implement the approved authorization context and explicit decisions for authoring, discovery, activation, planning, destination suitability, and runtime scope.
- Represent classification/retention constraints and reject unauthorized destinations, transformations, cleaning, telemetry, or retention.
- Emit redacted audit intent for decisions and denials.

Out of scope:
- Identity-provider or policy-product selection, UI-only enforcement, and permissive defaults.

Affected modules and contracts:
- Control-plane policy, planning boundary, scoped runtime authorization and audit categories.

Acceptance criteria:
- [ ] Given an unauthorized author/activator/planner or classified data sent to a prohibited destination, when the use case runs, then it denies the operation before side effects and records a redacted denial category.
- [ ] Given missing or indeterminate authorization/classification policy, when an action is evaluated, then it fails closed.

TDD first:
- First failing test: application test with a denied planning context asserts no run plan is persisted, no adapter is called, and one redacted denial audit intent is produced.
- Additional coverage: least privilege per scope, activation authorization, classification limits, telemetry redaction, policy timeout/cancellation.

Implementation notes:
- Implement only after BL-001.3 makes policy inputs concrete. Do not substitute local allow-all behavior.

Documentation updates:
- Functional: permissions, policy outcomes, and denial semantics.
- Installation: environment-specific identity/policy configuration after approval.
- Operations/release: access review, classification and retention procedure.
- Architecture/ADR: update boundary documentation if the approved decision changes responsibilities.

Security and data impact:
- Security-sensitive; require focused security review, audit, classification, retention/deletion, and least-privilege assessment.

Compatibility, migration, and rollback:
- Version policy/classification metadata; define effects on active revisions, migration, and rollback before enabling a new policy.

Environment verification:
- Local: deterministic policy doubles; no real identity credentials.
- Test: denied/allowed synthetic fixtures and audit assertions.
- Acceptance: independently managed non-production policy and classified-data scenarios.
- Production promotion or smoke: no policy deployment until approved; restricted denied-access and audit smoke is required later.

Dependencies:
- BL-001.3.
- BL-003.1.
- BL-004.1.
- BL-005.1.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, including focused security review.

## Parent coverage and blockers

Children cover opaque references/resolution, authorization decision points, classification, fail-closed denial, redaction, retention implications and required security testing. Both remain blocked on BL-001.3; no concrete identity or secret platform is selected.
