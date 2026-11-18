# BL-004 refinement — Create reusable contract, dependency, compatibility, and recovery tests

Parent: [BL-004](./backlog.md#bl-004-create-reusable-contract-dependency-compatibility-and-recovery-tests)  
Workflow: [`docs/implementation/change-workflow.md`](../implementation/change-workflow.md)

## Outcome and current evidence

**Current:** The repository has no tests, test fixtures, CI, or contract harness. **Required:** hermetic tests by default, reusable adapter conformance, synthetic data, and explicit recovery/compatibility coverage. **Open:** contract and version rules depend on BL-001.2.

Children establish reusable conformance/dependency checks, then shared fault/compatibility scenarios. They provide test infrastructure rather than product adapters.

Owner / target milestone for all children: **Open** — no owner or milestone is recorded in the repository; assign before moving any child to Ready.

Blocker tracking: For each **Blocked** child, its unresolved dependencies below are the blocker; the next action is to complete the first unresolved prerequisite in that list. Blocker owner is **Open** and must be assigned by maintainers.

## Child work items

### BL-004.1: Establish hermetic port-contract and dependency test harness

Type: Technical enabler  
Status: Blocked  
Priority: P0

Problem: Adapter substitution and module boundaries cannot be independently verified without shared tests.

Scope:
- Create reusable contract-test fixtures and runner conventions for repository, scheduler, engines, telemetry/audit, secret, source, destination, and state ports as contracts become available.
- Enforce architectural import boundaries and deterministic synthetic fixture conventions.

Out of scope:
- Implementing a reference adapter or selecting an external test service.

Affected modules and contracts:
- `testing/fixtures`, contract-test consumer surface, module dependency rules.

Acceptance criteria:
- [ ] Given two independent fake implementations of the same port, when the shared suite runs, then both are checked against the same observable behavior.
- [ ] Given a deliberately invalid inward dependency, when architecture tests run, then the test fails and identifies the importing boundary.
- [ ] Given a default test invocation, when run without network or credentials, then the suite is hermetic.

TDD first:
- First failing test: register a fake port implementation that violates one mandatory contract invariant and demonstrate the shared harness reports that named invariant.
- Additional coverage: fixture validation, context cancellation/timeout, unsupported capability, translated error, deterministic seed reporting.

Implementation notes:
- Build each suite only after the matching approved port exists; omit no required adapter category silently.

Documentation updates:
- Functional: N/A.
- Installation: clean test setup/teardown without undocumented global tools.
- Operations/release: document contract-test commands and interpreting failures.
- Architecture/ADR: record dependency rule in test ownership; follow BL-001.2 compatibility decisions.

Security and data impact:
- Synthetic data only; test output must redact secret-shaped values and reject production fixtures without approval.

Compatibility, migration, and rollback:
- Harness versioning must preserve reusable adapter-consumer API; no runtime-state migration.

Environment verification:
- Local: hermetic shared suite and dependency test.
- Test: same suite on every PR and for each adapter change.
- Acceptance: reuse applicable suites against acceptance adapter registrations.
- Production promotion or smoke: N/A; no production adapter mutation or credentials.

Dependencies:
- BL-002.1.
- BL-003.1.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus harness and fixture evidence above.

### BL-004.2: Add fault-injection, recovery, and concurrency contract scenarios

Type: Technical enabler  
Status: Blocked  
Priority: P0

Problem: Run correctness cannot be established by happy-path adapter tests alone.

Scope:
- Provide deterministic fault points for cancellation, timeout, duplicate delivery, stale worker, crash, commit ambiguity, checkpoint and acknowledgement boundaries, and recovery.
- Provide compatibility fixtures for previous/supported and unsupported contract versions after BL-001.2 policy is approved.

Out of scope:
- Defining the commit protocol or concurrency primitive; those are decisions in BL-009.

Affected modules and contracts:
- Recovery test harness, state/source/destination contract scenarios, migration fixtures.

Acceptance criteria:
- [ ] Given interruption before/after each configured durable boundary, when a scenario resumes, then observable state distinguishes committed, uncommitted, and ambiguous work.
- [ ] Given duplicate delivery or stale ownership, when the same scenario is rerun, then no newer evidence is overwritten and no duplicate committed result is accepted.
- [ ] Given an unsupported previous contract version, when compatibility validation runs, then it is rejected before partial migration or side effects.

TDD first:
- First failing test: simulate a crash after a destination commit but before checkpoint persistence; assert resume reconciles/deduplicates instead of blindly retrying.
- Additional coverage: stale-worker fencing, cancellation, timeout, idempotent retry, migration rollback, deterministic randomized seed.

Implementation notes:
- Use BL-009's approved durable boundaries; until then this item is blocked and scenarios must not encode a guessed protocol.

Documentation updates:
- Functional: explain tested recovery guarantees when implemented.
- Installation: document ephemeral integration setup and cleanup.
- Operations/release: recovery and compatibility evidence requirements.
- Architecture/ADR: link scenario semantics to approved BL-001.2 and BL-009 decisions.

Security and data impact:
- Synthetic records and fake credentials only; test redaction and denied-access scenarios where boundaries apply.

Compatibility, migration, and rollback:
- Test old/current/unsupported version behavior, migration atomicity, and rollback limits.

Environment verification:
- Local: deterministic fault suite with no ambient services.
- Test: run tagged integration/recovery suites against isolated ephemeral dependencies.
- Acceptance: verify selected recovery and migration scenarios against promoted candidate.
- Production promotion or smoke: N/A for harness; production smoke must be restricted and use synthetic-safe checks.

Dependencies:
- BL-001.2.
- BL-004.1.
- BL-003.2.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus the fault and compatibility evidence above.

## Parent coverage and blockers

Children cover shared adapter contract fixtures, dependency rules, synthetic data, recovery/fault scenarios, concurrency, compatibility and migration checks. BL-004.2 remains blocked until BL-009 chooses the valid durable boundary model.
