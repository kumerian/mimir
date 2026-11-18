# BL-011 refinement — Deliver scheduling, capability matching, and immutable run planning

Parent: [BL-011](./backlog.md#bl-011-deliver-scheduling-capability-matching-and-immutable-run-planning)  
Workflow: [`docs/implementation/change-workflow.md`](../implementation/change-workflow.md)

## Outcome and current evidence

**Current:** Local run plans pin active metadata, source/destination capabilities, ordered stages and rule versions. Fixed-clock schedule evaluation checks windows/dependencies and stores stable deduplicated requests; it does not dispatch runs. Cancellation/shutdown now writes immutable stop acknowledgements and audit evidence without changing run state. External scheduling remains absent.

Children separate eligibility/request semantics, then immutable plan/capability matching and replay preparation.

Owner / target milestone for all children: **Open** — no owner or milestone is recorded in the repository; assign before moving any child to Ready.

Blocker tracking: For each **Blocked** child, its unresolved dependencies below are the blocker; the next action is to complete the first unresolved prerequisite in that list. Blocker owner is **Open** and must be assigned by maintainers.

## Child work items

### BL-011.1: Evaluate in-process schedule eligibility and issue stable run requests

Type: Feature  
Status: In Progress
Priority: P0

Problem: An eligible trigger must request a run without owning ingestion logic or mutating execution state.

Scope:
- Implement in-process schedule/dependency/window/scope and retry eligibility using active metadata and policy.
- Issue stable request/run identities; handle duplicate trigger delivery, cancellation, timeout and shutdown.

Out of scope:
- External orchestrator, data execution and retrying an unsafe in-flight run.

Affected modules and contracts:
- Scheduler port/adapter, schedule metadata, request identity and planner handoff.

Acceptance criteria:
- [x] Given eligible and ineligible schedules, when evaluated at a fixed clock, then the scheduler emits only eligible stable requests with deterministic reasons.
- [x] Given duplicate delivery with the same request identity, when submitted repeatedly, then one logical request is produced without starting ingestion.
- [x] Given cancellation or shutdown, when requests are pending, then they are stopped/acknowledged according to policy without changing run execution state.

TDD first:
- First failing test: fixed-clock schedule test submits the same trigger twice and asserts a single stable request identity.
- Additional coverage: dependency graph, activation windows, retry/concurrency limits, cancellation, timeout and graceful shutdown.

Implementation notes:
- In-process scheduler is the accepted system's initial delivery shape; exact service-level and request-envelope decisions remain open.

Documentation updates:
- Functional: trigger/scope/dependency semantics and duplicate behavior.
- Installation: local scheduler configuration and bounded limits.
- Operations/release: clock, shutdown, backlog and retry diagnostics.
- Architecture/ADR: no scheduler-owned ingestion transitions; link BL-009.2 identity policy.

Security and data impact:
- Authorize trigger scope; avoid exposing classified metadata and secret values in request envelope.

Compatibility, migration, and rollback:
- Version stable request identity; do not replay requests across incompatible versions without explicit policy; no persisted schedule migration here.

Environment verification:
- Local: fixed-clock hermetic scheduler tests.
- Test: duplicate/cancellation contract suite in CI.
- Acceptance: trigger/dependency/load behavior against candidate with synthetic scope.
- Production promotion or smoke: no external scheduler or production trigger configured; later restricted health/request-id check only.

Dependencies:
- BL-005.2.
- BL-007.2.
- BL-009.2.
- BL-010.1.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus schedule contract and shutdown evidence above.

### BL-011.2: Match capabilities and persist immutable safe run plans

Type: Feature  
Status: In Progress
Priority: P0

Problem: A request must be rejected before side effects unless all pinned metadata, policies, adapters and delivery guarantees form a safe plan.

Scope:
- Implement `PlanRun`: load active revision, resolve effective policy, validate scope/dependencies, match declared source/destination/engine/state/security capabilities, select approved strategy, persist immutable plan and audit intent.
- Reject missing/changed/unsafe capabilities before execution.

Out of scope:
- Stage execution, external orchestration, unsafe retries, and guaranteeing replay beyond available source/destination capabilities.

Affected modules and contracts:
- Planner, run-plan aggregate, capability declarations, metadata/policy/repository/state/audit ports.

Acceptance criteria:
- [x] Given only active validated metadata and authorized scope with complete capabilities, when `PlanRun` runs, then the persisted immutable plan records revision, scope, stages, contract/rule versions and delivery guarantee.
- [x] Given draft metadata, denied scope, changed/missing capability, or unsupported commit combination, when planning runs, then it returns a stable error and no runnable plan or adapter side effect.
- [x] Given a repeated stable request, when planned, then the same logical run identity is returned without duplicate plan creation.

TDD first:
- First failing test: application test with a destination missing reconciliation capability asserts planning rejects before any adapter method or runnable plan persistence.
- Additional coverage: capability matrix, policy denial, activation race, plan immutability, idempotency, cancellation/timeout and schema compatibility.

Implementation notes:
- Choose only capability combinations approved in BL-009; no probing hidden adapter behavior.

Documentation updates:
- Functional: plan contents, planner errors, guarantees and examples.
- Installation: adapter registration and startup capability validation.
- Operations/release: limits, unsupported combinations, plan/revision inspection and shutdown.
- Architecture/ADR: plan immutability and scheduler separation.

Security and data impact:
- Enforce BL-005 classification/authorization; include references, never secret values.

Compatibility, migration, and rollback:
- Version run-plan contract; pin revision and adapter versions; test previous plan reading and reject incompatible versions without partial writes.

Environment verification:
- Local: hermetic application tests using controlled port doubles.
- Test: compatibility/dependency/capability tests in CI.
- Acceptance: representative safe and rejected plans on exact candidate.
- Production promotion or smoke: no production execution; later check plan contract/health with restricted scope.

Dependencies:
- BL-003.2.
- BL-004.1.
- BL-011.1.
- BL-007.3.
- BL-009.2.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus planner acceptance and audit evidence above.

### BL-011.3: Prepare replay plans only when pinned inputs remain available

Type: Feature  
Status: In Progress
Priority: P0

Problem: Replay must be explicit, linked to original execution, and honest about unavailable snapshots, positions, adapter versions, or rule versions.

Scope:
- Implement replay preparation from an original pinned plan using approved replay policy, source position/snapshot availability, contract/rule versions, and destination strategy.
- Record replay identity/reason and reject unavailable deterministic guarantees before execution.

Out of scope:
- Running the replay, reconstructing unavailable source history, or claiming deterministic replay for unsupported adapters.

Affected modules and contracts:
- Replay planning use case, immutable run plan, source/destination capabilities, metadata and audit.

Acceptance criteria:
- [x] Given all pinned source/version/strategy inputs remain available, when replay is prepared, then a new linked plan records original and replay identities and unchanged pinned metadata.
- [x] Given a missing/changed required position, adapter contract or rule version, when replay is prepared, then it is rejected or explicitly marked non-deterministic according to approved policy before side effects.

TDD first:
- First failing test: replay fixture changes the source position capability and asserts planning refuses a deterministic replay without creating a runnable plan.
- Additional coverage: same-input plan identity, reason/audit, unsupported compatibility, cancellation/timeout and destination duplicate/reconciliation policy.

Implementation notes:
- Replay guarantees are bounded by BL-009.2 and source/destination profile; never substitute current metadata silently.

Documentation updates:
- Functional: replay eligibility, linked identities, guarantees and limitations.
- Installation: no separate install; required adapter versions and local fixtures.
- Operations/release: replay audit, denial and reconciliation runbook.
- Architecture/ADR: state replay availability policy and version pinning.

Security and data impact:
- Replay requires reauthorization and classification checks; secret references remain opaque.

Compatibility, migration, and rollback:
- Retain original plan/history; do not mutate it. Reject unsupported plan versions; rollback cancels only the new replay identity under approved state rules.

Environment verification:
- Local: synthetic source-position/version fixtures.
- Test: replay preparation suite in CI.
- Acceptance: verify available/unavailable replay scenarios against candidate.
- Production promotion or smoke: no production replay; later restricted smoke is read-only and must not trigger data writes.

Dependencies:
- BL-011.2.
- BL-009.2.
- BL-010.2.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus replay safety and audit evidence above.

## Parent coverage and blockers

Children cover in-process schedule/dependency/retry/scope, stable duplicate-safe requests, active-revision capability matching, immutable plan persistence, and conservative replay preparation. Actual execution/retry remains BL-012 onward.
