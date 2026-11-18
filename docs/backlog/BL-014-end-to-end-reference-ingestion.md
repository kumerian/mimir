# BL-014 refinement — Deliver an end-to-end safe reference ingestion path

Parent: [BL-014](./backlog.md#bl-014-deliver-an-end-to-end-safe-reference-ingestion-path)  
Workflow: [`docs/implementation/change-workflow.md`](../implementation/change-workflow.md)

## Outcome and current evidence

**Current:** There is no executable application, CLI, health check, run, state, audit, or data path. **Required:** one-process composition must demonstrate validated metadata → immutable plan → bounded raw ingestion → durable commit → checkpoint/ack → run state and audit correlation. **Open:** command surface and all concrete reference profiles remain undecided.

Children provide composition/local entry, a successful end-to-end slice, then restart/fault/acceptance documentation. Transformations, quality, cleaning, API, external worker and production scale are excluded as parent specifies.

Owner / target milestone for all children: **Open** — no owner or milestone is recorded in the repository; assign before moving any child to Ready.

Blocker tracking: For each **Blocked** child, its unresolved dependencies below are the blocker; the next action is to complete the first unresolved prerequisite in that list. Blocker owner is **Open** and must be assigned by maintainers.

## Child work items

### BL-014.1: Compose approved modules behind a local command and health boundary

Type: Feature  
Status: Blocked  
Priority: P0

Problem: Independently tested modules need a single-process composition path that wires only approved adapters and exposes basic readiness without embedding business transitions.

Scope:
- Compose metadata control plane, scheduler, planner, coordinator, state/audit/telemetry and reference source/destination from approved capability registrations.
- Add thin local CLI/local command and health/readiness/build-contract information; fail startup on missing/inconsistent required capabilities.

Out of scope:
- Public API, separately deployed worker, production deployment product, transformations or data execution beyond a composition smoke.

Affected modules and contracts:
- Entrypoint/composition, CLI/local command, health/readiness, adapter registry and application use cases.

Acceptance criteria:
- [ ] Given valid approved local configuration, when the command starts, then it reports ready with build/contract identity and does not expose secrets.
- [ ] Given missing/changed required capability or invalid configuration, when startup runs, then it fails closed before accepting a run.
- [ ] Given a user command, when dispatched, then entrypoint translates input and calls a use case without performing business state transitions itself.

TDD first:
- First failing test: composition test omits a required capability and asserts startup returns a stable configuration error without opening source/destination.
- Additional coverage: CLI arguments/output/exit codes, health/readiness, configuration version, shutdown and redaction.

Implementation notes:
- Public command naming is open; do not claim this is the final CLI contract absent approval.

Documentation updates:
- Functional: local command purpose, health output and failure behavior.
- Installation: clean local setup, approved profile, config, validation and cleanup.
- Operations/release: startup diagnostics, readiness, shutdown, capabilities and build identity.
- Architecture/ADR: update composition boundary only if approved structure changes.

Security and data impact:
- Apply control/planning/runtime policy; never emit secret values; local data is synthetic and constrained.

Compatibility, migration, and rollback:
- Version config/CLI output; reject unsupported config before side effects; no persisted data migration in composition. Roll back by restoring prior candidate/config.

Environment verification:
- Local: clean install, command smoke, health, config failure and cleanup.
- Test: candidate composition and hermetic test suite in CI.
- Acceptance: configure exact candidate against isolated non-production profile.
- Production promotion or smoke: production deployment not in scope; reserve restricted health/build-identity smoke for BL-020 approval.

Dependencies:
- BL-010.1.
- BL-011.1.
- BL-012.1.
- BL-013.1.
- BL-002.1.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus composition/security/install evidence above.

### BL-014.2: Complete a bounded synthetic raw-ingestion run with correlated outcomes

Type: Feature  
Status: Blocked  
Priority: P0

Problem: Mimir needs an observable end-to-end success path proving control-plane, planning, data movement and state/audit ordering work together.

Scope:
- Execute one bounded synthetic source scope using validated/activated metadata, stable request, immutable plan, approved source/destination, run coordinator and state/audit/telemetry.
- Preserve raw input identity/provenance and correlate destination, checkpoint, run/attempt, audit and telemetry.
- Enforce commit-before-checkpoint-before-source-ack ordering where supported.

Out of scope:
- Transformation, quality, cleaning, production data, unsupported replay claims, public API and external worker.

Affected modules and contracts:
- End-to-end use cases, local command, source/destination ports, run state/checkpoint, audit/telemetry and lineage.

Acceptance criteria:
- [ ] Given valid synthetic metadata and approved capabilities, when a local run executes, then it reaches Succeeded only after destination durable commit, valid checkpoint, applicable acknowledgement, and required audit evidence.
- [ ] Given the same synthetic run identity is delivered twice, when both requests execute, then one logical result is committed and identities correlate across state, destination, audit and telemetry.
- [ ] Given invalid metadata, denied destination, missing capability, or ambiguous commit, when execution is attempted, then unsafe publication is prevented or left explicitly unresolved, never reported as success.

TDD first:
- First failing test: end-to-end synthetic acceptance test asserts identical run/attempt/batch identity across destination commit evidence, checkpoint, audit and telemetry after successful execution.
- Additional coverage: invalid metadata, permission denial, duplicate request, cancellation, timeout, ambiguous commit and no-partial-success assertions.

Implementation notes:
- Preserve raw values and source provenance; do not add transformations or cleaning to the first slice.

Documentation updates:
- Functional: first-run guide, prerequisites, inputs/outputs, state transitions, guarantees, failure/retry/recovery, audit and limitations.
- Installation: supported local target, integrity, configuration, secret references, permissions, validate/run/cleanup.
- Operations/release: readiness, correlation, backup/restore, limits, failure/reconciliation, rollback and known limitations.
- Architecture/ADR: retain stage order and mark implemented behavior separately from intended later capability.

Security and data impact:
- Synthetic data by default; authorize each boundary, classify test data, redact secrets and sensitive payloads, audit access/activation/commit.

Compatibility, migration, and rollback:
- Verify metadata, plan, adapter and state versions; no production data migration. On rollback retain commit evidence and do not resume with incompatible checkpoint semantics.

Environment verification:
- Local: clean install and deterministic synthetic run with cleanup.
- Test: end-to-end CI plus duplicate/failure suite on isolated resources.
- Acceptance: full synthetic sign-off against exact candidate artifact and approved profile.
- Production promotion or smoke: BL-014 does not authorize promotion; after BL-020, restricted non-mutating health/contract smoke only unless separately approved.

Dependencies:
- BL-014.1.
- BL-010.2.
- BL-010.3.
- BL-011.2.
- BL-012.2.
- BL-013.2.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, including security, recovery, independent review and acceptance evidence.

### BL-014.3: Prove restart recovery and publish first-slice operating/install guidance

Type: Feature  
Status: Blocked  
Priority: P0

Problem: A happy-path run is insufficient; operators need evidence that interruptions do not duplicate or lose committed work and instructions to recover safely.

Scope:
- Exercise supported crash/restart points around destination commit, checkpoint and acknowledgement, duplicate delivery, cancellation and shutdown.
- Publish verified end-to-end functional, clean installation, operational recovery, limitations and candidate artifact test evidence.

Out of scope:
- Production-scale recovery, unsupportable replay scenarios, and deployment automation owned by BL-020.

Affected modules and contracts:
- BL-014 end-to-end harness, run state, destination reconciliation, checkpoint, CLI and operations docs.

Acceptance criteria:
- [ ] Given every BL-009.2-approved crash boundary, when the process restarts, then it resumes, deduplicates, reconciles, or remains explicitly blocked without success-shaped ambiguity.
- [ ] Given the documented clean installation, when executed on each approved local target, then a reviewer can run and clean the synthetic scenario without production accounts or undocumented global tools.
- [ ] Given the candidate artifact, when acceptance evidence is recorded, then the artifact digest, source revision, configuration/metadata versions and test results are linked.

TDD first:
- First failing test: fault-injection test crashes after durable destination commit before checkpoint and asserts restart reconciliation produces exactly one committed result.
- Additional coverage: each supported boundary, stale worker, source acknowledgement, cancellation, CLI smoke, cleanup, doc-link/schema/example checks.

Implementation notes:
- The approved reference profile defines test boundaries; unsupported recovery paths must be documented as unavailable, not simulated as guaranteed.

Documentation updates:
- Functional: first-run/outputs/failure/recovery examples with synthetic data.
- Installation: clean setup, versions, integrity, config, permissions, health, upgrade/rollback/cleanup.
- Operations/release: restart/reconciliation runbook, audit/telemetry, backups, known limitations and acceptance digest.
- Architecture/ADR: update Current implementation status only after verification; proposed architecture remains distinct.

Security and data impact:
- No production payloads or credentials; verify redaction, least privilege and cleanup of synthetic data.

Compatibility, migration, and rollback:
- Verify supported previous metadata/state contract and reject unsupported versions before side effects; preserve recovery evidence and prior candidate for rollback.

Environment verification:
- Local: repeated clean installation and crash/restart scenario.
- Test: full fault matrix and docs/schema/link validators in CI.
- Acceptance: sign off recovery and installation using the same immutable candidate artifact.
- Production promotion or smoke: no promotion in this child; production smoke requires BL-020 and explicit approval, using restricted scope.

Dependencies:
- BL-014.2.
- BL-004.2.
- BL-002.2.
- BL-009.2.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus recovery, docs and exact-artifact acceptance evidence above.

## Parent coverage and blockers

Children cover one-process composition, local command/health, a bounded raw run, identity/provenance, state/audit/telemetry correlation, safe commit/checkpoint/ack ordering, failure prevention, restart recovery, clean install, operations guide, and acceptance evidence. Parent exclusions remain excluded. All children are Blocked by unresolved BL-001/BL-009 decisions and their implementation dependencies; none are Ready.
