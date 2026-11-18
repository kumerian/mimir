# BL-013 refinement — Deliver source and destination reference adapters with safe commit contracts

Parent: [BL-013](./backlog.md#bl-013-deliver-source-and-destination-reference-adapters-with-safe-commit-contracts)  
Workflow: [`docs/implementation/change-workflow.md`](../implementation/change-workflow.md)

## Outcome and current evidence

**Current:** Local JSONL source and append-only destination adapters implement the versioned ports and pass shared source/destination conformance tests. Snapshot identity, idempotent commit, reconciliation, durable evidence, timeout/cancellation mapping, denied scope, and unsupported acknowledgement are tested; third-party adapters remain deferred.

Children cover source reads, destination commit/reconciliation, and shared conformance/integration verification. They do not select products.

Owner / target milestone for all children: **Open** — no owner or milestone is recorded in the repository; assign before moving any child to Ready.

Blocker tracking: For each **Blocked** child, its unresolved dependencies below are the blocker; the next action is to complete the first unresolved prerequisite in that list. Blocker owner is **Open** and must be assigned by maintainers.

## Child work items

### BL-013.1: Implement the approved bounded-batch reference source adapter

Type: Feature  
Status: In Progress
Priority: P0

Problem: A run needs a deterministic source adapter that exposes exactly its declared scope, identity and position guarantees.

Scope:
- Implement the BL-009.3-approved reference source with declared versioned capabilities, bounded batch scope, stable record/batch identity, source position, schema observation if selected, acknowledgement if selected, cancellation/timeout and stable error translation.
- Keep adapter behavior behind source port.

Out of scope:
- Streaming support, external provider selection, destination writes or hidden retries.

Affected modules and contracts:
- Source port, reference source adapter, capability declaration, source record/provenance and secret resolver.

Acceptance criteria:
- [x] Given a bounded synthetic scope, when the source is read twice under the same approved snapshot/identity guarantee, then records and positions match the declared contract. The reusable source contract reads the same snapshot twice and compares identities/records.
- [x] Given unsupported scope, missing capability, cancellation, timeout or source failure, when operation runs, then it returns a stable categorized failure without claiming successful acknowledgement. Local tests cover denied scope, cancellation, timeout, unavailable source, planner capability rejection, and the explicit unsupported acknowledgement capability.

TDD first:
- First failing test: shared source contract supplies the same synthetic batch twice and asserts stable identity/position semantics from the approved profile.
- Additional coverage: capability, partial reads, timeout/cancellation, schema behavior, denied access, redaction and compatibility.

Implementation notes:
- No source format/product is approved yet; implementation cannot start until BL-009.3 is decided.

Documentation updates:
- Functional: declared source scope/position/guarantees and limitations.
- Installation: local/test setup and cleanup after approval; no production credentials.
- Operations/release: source errors, limits, timeout and diagnostics.
- Architecture/ADR: preserve normalized source contract; provider-specific detail stays in adapter.

Security and data impact:
- Scope source access least-privilege; protect sensitive observations and values; only references for credentials.

Compatibility, migration, and rollback:
- Version adapter contract/capability; preserve source position meanings across upgrade or reject incompatible resume; no destructive source changes.

Environment verification:
- Local: deterministic synthetic source and contract suite.
- Test: isolated adapter integration and failure injection.
- Acceptance: verify approved capability limits on candidate against non-production data.
- Production promotion or smoke: no production source authorized by reference decision; later restricted read-only smoke and approval required.

Dependencies:
- BL-003.2.
- BL-004.1.
- BL-009.3.
- BL-011.2.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, including adapter security and contract review.

### BL-013.2: Implement destination idempotency, durable commit, and reconciliation

Type: Feature  
Status: In Progress
Priority: P0

Problem: The destination must expose durable evidence and reconcile uncertain outcomes before a run can safely checkpoint or retry.

Scope:
- Implement the BL-009.3-approved destination's capability declaration, target validation, staged/idempotent writes, stable operation identities, durable commit evidence, reconciliation, timeout/cancellation and stable error translation.
- Preserve classification and secret boundaries.

Out of scope:
- Assuming atomic cross-system commits, claiming exactly-once without evidence, or adding an unapproved destination.

Affected modules and contracts:
- Destination port/adapter, commit outcome, idempotency key, reconciliation and checkpoint integration.

Acceptance criteria:
- [x] Given duplicate delivery of a committed batch, when the approved idempotency strategy applies, then the observable committed result has no duplicate records.
- [x] Given uncertain commit state, when reconcile is requested, then outcome is established or remains explicitly ambiguous; the adapter never silently reports success or blindly retries. Run recovery tests crash after durable commit and reconcile by stable identity before checkpointing.
- [x] Given crash before/after commit, when operation resumes, then returned evidence allows BL-012 checkpoint policy to decide safely.

TDD first:
- First failing test: fault-injection test commits a synthetic batch but drops the response; retry path first reconciles and yields one committed result.
- Additional coverage: duplicate write, crash matrix, staging cleanup, timeout/cancellation, unsupported capability, authorization, error mapping and compatibility.

Implementation notes:
- Exact semantics depend on BL-009.2/.3; do not create an adapter-specific guarantee absent approved evidence.

Documentation updates:
- Functional: delivery guarantee, reconciliation and ambiguous outcome.
- Installation: approved local/test destination setup and cleanup.
- Operations/release: backup, target validation, recovery, limits and runbook.
- Architecture/ADR: document capability mapping and non-guarantees.

Security and data impact:
- Enforce destination classification/authorization and least privilege; redact payloads and secret values.

Compatibility, migration, and rollback:
- Version destination contract and committed identity; define data cleanup/forward recovery; rollback cannot erase evidence of possibly committed writes.

Environment verification:
- Local: deterministic reference profile and idempotency tests.
- Test: isolated integration with injected commit faults.
- Acceptance: replay/reconciliation scenarios on exact candidate with synthetic data.
- Production promotion or smoke: no production writes by default; any future test requires explicit restricted target and approval.

Dependencies:
- BL-009.3.
- BL-009.2.
- BL-003.2.
- BL-004.2.
- BL-012.2.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, including adapter security and recovery review.

### BL-013.3: Verify adapters through reusable contract and integration suites

Type: Technical enabler  
Status: In Progress
Priority: P0

Problem: Reference adapters must demonstrate substitutability and contract-defined failure behavior, not merely pass implementation-specific happy paths.

Scope:
- Run shared source/destination conformance tests, capability completeness, stable error translation, secret/classification redaction, and adapter-specific isolated integration tests.
- Publish adapter setup, cleanup and supported/unsupported behavior.

Out of scope:
- Adding other providers or changing the approved contract to fit an adapter.

Affected modules and contracts:
- BL-004 harness, BL-013.1/.2 adapters, capability manifests and integration setup.

Acceptance criteria:
- [x] Given either reference adapter, when the same shared port suite runs, then it verifies the same externally observable contract assertions.
- [x] Given a deliberately omitted capability, when the test matrix runs, then it is declared and planner rejection is tested rather than silently skipped.
- [x] Given adapter timeout, cancellation, denied access, and adapter error, when mapped, then stable categories and redacted details are asserted.

TDD first:
- First failing test: register a reference adapter with a missing capability field and assert the capability-conformance suite fails before adapter acceptance.
- Additional coverage: source/destination contract, crash, duplicate, migration, cancellation, timeout, redaction and integration cleanup.

Implementation notes:
- Do not add product-specific conditions to shared contract tests.

Documentation updates:
- Functional: adapter guarantee matrix and limitations.
- Installation: exact approved local/test setup, integrity, cleanup and troubleshooting.
- Operations/release: contract certification evidence and upgrade/rollback.
- Architecture/ADR: link to adapter profile and versions.

Security and data impact:
- Use only synthetic data and fake credentials in default test suite; require isolated least-privilege integration credentials when needed.

Compatibility, migration, and rollback:
- Run previous supported port-version fixtures and explicitly reject unsupported versions; document adapter downgrade constraints.

Environment verification:
- Local: hermetic contract suite.
- Test: tagged integration tests against ephemeral/isolated resources.
- Acceptance: exact candidate adapter certification and recovery scenarios.
- Production promotion or smoke: no production writes; use restricted non-mutating smoke after explicit approval.

Dependencies:
- BL-013.1.
- BL-013.2.
- BL-004.1.
- BL-004.2.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus shared contract and integration evidence above.

## Parent coverage and blockers

Children cover source read/identity/position, destination staged idempotent commit and reconciliation, full declared capabilities, error/security behavior, and reusable adapter conformance/integration. Each product and delivery guarantee is blocked until BL-009.2/.3 decisions are approved.
