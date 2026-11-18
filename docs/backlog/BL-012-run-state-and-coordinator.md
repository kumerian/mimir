# BL-012 refinement — Deliver run, attempt, checkpoint, fencing, and coordinator semantics

Parent: [BL-012](./backlog.md#bl-012-deliver-run-attempt-checkpoint-fencing-and-coordinator-semantics)  
Workflow: [`docs/implementation/change-workflow.md`](../implementation/change-workflow.md)

## Outcome and current evidence

**Current:** The local runtime appends lifecycle events through a conditional file-backed state port, uses run identity/owner tokens, records attempt/failure-stage history, reconciles destination and quarantine evidence, and rejects checkpoint-only success. Local tests cover retries, stale owners, cancellation, and before/after supported crash boundaries. Distributed worker fencing and an independently deployed worker shutdown protocol remain.

Children deliver (1) lifecycle/history/fencing, (2) checkpoint validity and reconciliation, and (3) bounded stage coordination and shutdown.

Owner / target milestone for all children: **Open** — no owner or milestone is recorded in the repository; assign before moving any child to Ready.

Blocker tracking: For each **Blocked** child, its unresolved dependencies below are the blocker; the next action is to complete the first unresolved prerequisite in that list. Blocker owner is **Open** and must be assigned by maintainers.

## Child work items

### BL-012.1: Enforce run lifecycle, attempt history, and fenced ownership

Type: Technical enabler  
Status: In Progress
Priority: P0

Problem: Duplicate or stale workers must not overwrite newer lifecycle evidence or erase prior failures.

Scope:
- Implement run/attempt states, allowed transitions, immutable history, owner acquisition/fencing, retry as new attempt or linked replay, terminal failures and cancellation.
- Persist through the run-state port with conditional version checks.

Out of scope:
- Destination publication, stage business logic, or assuming a specific concurrency backend.

Affected modules and contracts:
- Run/attempt aggregate, transition policy, state store, owner/fencing token and stable failure history.

Acceptance criteria:
- [x] Given an invalid transition or stale owner token, when the state store receives it, then it rejects the transition and preserves the latest state/evidence.
- [x] Given a failed attempt retried, when the next attempt starts, then the prior attempt/failure/commit evidence remains immutable and linked. Run-state events retain attempt, owner, stable category, and failure stage.
- [x] Given duplicate run request identity, when ownership is acquired concurrently, then only one current owner is accepted. File run-state append is serialized and conflicting initial owners are rejected; the local profile is single-process.

TDD first:
- First failing test: race an old and new fencing token against the same run and assert the stale worker cannot advance or overwrite checkpoint/state.
- Additional coverage: all valid/invalid transitions, duplicate delivery, retry history, cancellation, timeout and persistence restart.

Implementation notes:
- Use BL-009.2 ownership/fencing decision and BL-010.1 state adapter contract.

Documentation updates:
- Functional: run lifecycle, attempts, terminal states and retry behavior.
- Installation: worker limits and required state-store registration.
- Operations/release: stale owner detection, shutdown, backup and recovery.
- Architecture/ADR: preserve single-writer invariant and state/data separation.

Security and data impact:
- Authorize run operations; state stores secret references only and redact sensitive failure details.

Compatibility, migration, and rollback:
- Version persisted states; retain prior attempts; test compatible read/upgrade and unsupported state rejection. Do not erase history on rollback.

Environment verification:
- Local: hermetic transition/race tests.
- Test: race and persistence integration tests with synthetic runs.
- Acceptance: restart and stale-worker recovery on the exact candidate.
- Production promotion or smoke: only after deployment approval; restricted fencing/health smoke before traffic.

Dependencies:
- BL-004.2.
- BL-009.2.
- BL-010.1.
- BL-011.2.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus concurrency and persistence evidence above.

### BL-012.2: Validate checkpoints and reconcile ambiguous durable outcomes

Type: Technical enabler  
Status: In Progress
Priority: P0

Problem: Resumption is safe only when checkpoint evidence proves corresponding destination work is durable and duplicate work is handled.

Scope:
- Implement checkpoint identity/validity and source/destination ordering from BL-009.2.
- Record durable commit evidence before checkpoint, reconcile unknown commits, and acknowledge source position only after valid checkpoint where adapter supports acknowledgement.

Out of scope:
- Choosing commit semantics or destination technology; those are BL-009/013.

Affected modules and contracts:
- Checkpoint value/state persistence, destination commit result, source position/acknowledgement and recovery outcome.

Acceptance criteria:
- [x] Given no durable destination commit evidence, when a checkpoint is proposed, then it is rejected.
- [x] Given an ambiguous commit, when resume begins, then reconciliation is required before retry and the run cannot report success until outcome is known.
- [x] Given durable commit evidence, when checkpoint persists and source supports acknowledgement, then order is commit, checkpoint, acknowledgement. The local source declares no acknowledgement capability; local ordering is durable commit then checkpoint and audit.

TDD first:
- First failing test: inject a crash after destination commit and before checkpoint; on restart assert reconciliation/deduplication happens before any source acknowledgement.
- Additional coverage: crash at every durable boundary, stale checkpoint, duplicate batch, no-ack source, timeout, cancellation and unsupported strategy rejection.

Implementation notes:
- Use approved commit/checkpoint strategy only; do not promise exactly-once delivery without contract evidence.

Documentation updates:
- Functional: checkpoint/recovery guarantees and unsupported scenarios.
- Installation: state-store and adapter capability prerequisites.
- Operations/release: ambiguous commit reconciliation and manual hold procedures.
- Architecture/ADR: keep checkpoint rule aligned to BL-009.2 and BL-013.

Security and data impact:
- Record identities and references only; no raw sensitive payload or resolved secret in checkpoint.

Compatibility, migration, and rollback:
- Version checkpoint format/granularity; migrate only under approved strategy with provenance; rollback preserves prior evidence and blocks unsafe resume.

Environment verification:
- Local: deterministic in-memory fault injection.
- Test: contract/integration crash matrix with isolated dependencies.
- Acceptance: recover at each supported boundary on exact candidate.
- Production promotion or smoke: later restricted recovery/fencing smoke; no production data replay as validation.

Dependencies:
- BL-012.1.
- BL-009.2.
- BL-010.1.
- BL-004.2.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus durable-boundary and recovery evidence above.

### BL-012.3: Coordinate bounded execution stages and graceful interruption

Type: Technical enabler  
Status: In Progress
Priority: P0

Problem: The runtime needs one owner to sequence approved stages, preserve state and stop predictably on cancellation, timeout, failure or shutdown.

Scope:
- Implement stage command/result boundaries and coordinator for approved plan order; persist progress/failure through state port and audit intents.
- Bound in-flight work, propagate context cancellation/deadlines, and checkpoint/release ownership on graceful shutdown only when safe.

Out of scope:
- Implementing source/destination adapters or transformation/quality/cleaning behavior.

Affected modules and contracts:
- Execution coordinator, stage interfaces, run-state transitions, audit and telemetry correlation.

Acceptance criteria:
- [x] Given a stage failure, when the coordinator executes, then later stages do not run and run/attempt state records the stable failure category and stage. Failures before publication remain uncommitted; terminal events persist category/stage.
- [x] Given cancellation or shutdown, when bounded work is active, then it stops accepting work, drains or safely checkpoints by policy, releases ownership, and never reports success without durable terminal evidence. Local cancellation is tested during processing; no independent worker/shutdown protocol is implemented.
- [x] Given a stale worker resumes, when it tries to write, then fencing rejects it. A newer attempt's owner token fences prior events within the single-host local profile; distributed leases are unsupported.

TDD first:
- First failing test: controlled stage double blocks until cancellation; assert coordinator stops downstream stages and records deterministic cancelled state.
- Additional coverage: stage ordering, timeout, panic/error mapping, audit failure, duplicate request, shutdown and correlation.

Implementation notes:
- Stage data values are immutable; stage executors do not reach into other stages' persistence.

Documentation updates:
- Functional: stage order, errors and cancellation.
- Installation: worker limits, startup validation and graceful shutdown configuration.
- Operations/release: draining, timeout, stuck stage and recovery procedures.
- Architecture/ADR: document only if an approved boundary changes.

Security and data impact:
- Pass only scoped credentials/data; redact stage errors, propagate classification and maintain audit evidence.

Compatibility, migration, and rollback:
- Version stage command/result contracts and plans; reject incompatible plans before starting. Rollback must not resume with incompatible checkpoint semantics.

Environment verification:
- Local: hermetic controlled stage tests.
- Test: concurrency/race and crash tests with synthetic data.
- Acceptance: cancellation, restart, shutdown and recovery against candidate.
- Production promotion or smoke: require approved worker packaging; restricted shutdown/fencing smoke before enabling traffic.

Dependencies:
- BL-012.2.
- BL-010.2.
- BL-011.2.
- BL-004.2.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus coordinator and recovery evidence above.

## Parent coverage and blockers

Children cover run/attempt lifecycle, immutable history, fencing, checkpoint validity, commit reconciliation, coordinator sequencing, retry/cancel/timeout/shutdown and crash recovery. Transform and quality stages remain separate future increments. All runtime implementation is blocked on BL-009/010 decisions.
