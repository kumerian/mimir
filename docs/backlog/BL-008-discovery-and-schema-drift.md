# BL-008 refinement — Deliver discovery and controlled schema-drift proposals

Parent: [BL-008](./backlog.md#bl-008-deliver-discovery-and-controlled-schema-drift-proposals)  
Workflow: [`docs/implementation/change-workflow.md`](../implementation/change-workflow.md)

## Outcome and current evidence

**Current:** No discovery provider, normalized observation, or schema-drift logic exists. **Required:** discovery creates proposals and never mutates active metadata; drift handling is explicit and policy-governed. **Open:** BL-009 selects any reference provider only after capability needs are clear.

The children separate provider observation/provenance from drift classification and proposal handoff. Together they cover BL-008 without selecting a source technology.

Owner / target milestone for all children: **Open** — no owner or milestone is recorded in the repository; assign before moving any child to Ready.

Blocker tracking: For each **Blocked** child, its unresolved dependencies below are the blocker; the next action is to complete the first unresolved prerequisite in that list. Blocker owner is **Open** and must be assigned by maintainers.

## Child work items

### BL-008.1: Capture normalized discovery observations as proposed metadata

Type: Feature  
Status: Blocked  
Priority: P1

Problem: Metadata owners need a reproducible view of observed source structure without granting discovery power to modify the active contract.

Scope:
- Implement discovery use case through the declared discovery-provider port; validate its capability before inspecting data.
- Normalize observations, record provider/version/scope/time provenance, and create a proposed candidate linked to—but distinct from—the active revision.

Out of scope:
- Selecting a concrete provider, automatic activation, or modifying source data.

Affected modules and contracts:
- Discovery use case/provider port, normalized observation, proposed metadata revision and audit intent.

Acceptance criteria:
- [ ] Given a provider with required discovery capability, when discovery runs on a synthetic source, then normalized observations and provenance are persisted as a proposal while active revision remains unchanged.
- [ ] Given missing capability, timeout, cancellation, or partial observation, when discovery runs, then it returns a classified failure/partial result and performs no active-metadata update.

TDD first:
- First failing test: application test invokes discovery with a fake provider and asserts a proposal exists while active revision identity is byte-for-byte unchanged.
- Additional coverage: provider contract, capability rejection before call, partial result, timeout/cancellation, redaction and duplicate request.

Implementation notes:
- Keep observations technology-neutral; any source-specific details belong to provider extensions with declared versions.

Documentation updates:
- Functional: discovery permissions, normalized outputs, proposal workflow and synthetic examples.
- Installation: provider registration/configuration only once BL-009 approves a profile.
- Operations/release: timeouts, partial results, diagnostics and cleanup.
- Architecture/ADR: keep discovery separate from authoring and activation.

Security and data impact:
- Authorize source inspection; enforce scope/classification and exclude sensitive values from schema observations and logs.

Compatibility, migration, and rollback:
- Version observations/proposals and retain provider provenance; rollback discards/retires proposal, never rewrites active history.

Environment verification:
- Local: deterministic fake provider and synthetic metadata.
- Test: shared discovery contract and policy tests in CI.
- Acceptance: isolated non-production schema observation, no production writes.
- Production promotion or smoke: no live discovery until access policy/provider approved; restricted read-only check later.

Dependencies:
- BL-003.2.
- BL-004.1.
- BL-005.2.
- BL-007.1.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, including data-access/security review.

### BL-008.2: Classify schema drift and route proposals through controlled validation

Type: Feature  
Status: Blocked  
Priority: P1

Problem: Schema changes need a defined disposition and review path rather than silently broadening active metadata.

Scope:
- Compare normalized observation to pinned active revision; classify additions, removals, type/constraint changes, compatibility and affected scope.
- Implement explicit reject, hold, quarantine, or continue outcomes as approved policy and submit candidate through BL-006/BL-007 validation/activation path.

Out of scope:
- Automatically activating proposals, inferring transformations, or applying unapproved drift policy.

Affected modules and contracts:
- Drift classifier, metadata compatibility report, proposal workflow, validation/activation ports.

Acceptance criteria:
- [ ] Given each configured drift class and policy, when an observation is compared, then the same input yields a deterministic disposition, affected scope, and reason.
- [ ] Given an incompatible or held change, when proposal processing completes, then active revision is unchanged and activation is blocked with an explainable diagnostic.
- [ ] Given a compatible proposal, when it is submitted, then activation still requires the separate authorized BL-007 path.

TDD first:
- First failing test: fixture for an incompatible column type change asserts it is classified and cannot change the active revision.
- Additional coverage: additive/removal/type/constraint fixtures, hold/reject/quarantine/continue policies, partial observations, concurrent activation.

Implementation notes:
- “Continue” can permit a run only under an explicit policy; it cannot rewrite metadata or conceal incompatibility.

Documentation updates:
- Functional: drift categories, policy outcomes, review/activation and examples.
- Installation: no new product; provider and policy configuration follows approved reference profile.
- Operations/release: triage, hold/reject recovery and audit evidence.
- Architecture/ADR: document policy behavior and update architecture only if decision changes.

Security and data impact:
- Protect schema/classification details; authorize observation access and redact sensitive fields.

Compatibility, migration, and rollback:
- Compare versions before activation; retain old revision and proposal; no silent metadata migration.

Environment verification:
- Local: table-driven synthetic drift scenarios.
- Test: compatibility and activation-race suite in CI.
- Acceptance: verify all configured dispositions against candidate artifact.
- Production promotion or smoke: no production schema writes; any later smoke is read-only and audited.

Dependencies:
- BL-008.1.
- BL-006.2.
- BL-007.2.
- BL-005.2.
- BL-001.2.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus drift/compatibility evidence above.

## Parent coverage and blockers

Children cover normalized, provenance-bearing observations, declared-provider capability checks, drift classification, compatibility and affected scope, explicit policy dispositions, and the no-auto-activation invariant. The first provider/product remains open under BL-009.
