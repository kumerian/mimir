# BL-001 refinement — Approve foundation decisions and ADR set

Parent: [BL-001](./backlog.md#bl-001-approve-foundation-decisions-and-adr-set)  
Workflow: [`docs/implementation/change-workflow.md`](../implementation/change-workflow.md)

## Outcome and current evidence

**Current:** This repository contains architecture and agent documentation, but no source, tests, ADRs, module, CI, or deployment artifacts. The system architecture is Accepted; the software architecture and technology baseline are Proposed. The backlog blocker register records approval, compatibility, security, and delivery decisions as open.

**Required:** Decision records state context, decision, consequences, alternatives, status, owner, and implications; no dependent implementation silently chooses a platform product. This refinement covers BL-001 by making its approval gates independently reviewable. The children below are **Proposed**, not approved architecture decisions.

Owner / target milestone for all children: **Open** — no owner or milestone is recorded in the repository; assign before moving any child to Ready.

Blocker tracking: For each **Blocked** child, its unresolved dependencies below are the blocker; the next action is to complete the first unresolved prerequisite in that list. Blocker owner is **Open** and must be assigned by maintainers.

## Child work items

### BL-001.1: Approve the language, package, and local build baseline

Type: Architecture decision  
Status: Proposed  
Priority: P0

Problem: The proposed Go/module/tooling baseline is not yet accepted, so implementation and reproducible-build claims have no approved target.

Scope:
- Decide whether to accept or revise Go, module layout, supported Go floor and OS/architecture matrix, task runner, linter/static-analysis tools, schema tooling, and local development baseline.
- Record options, owner, decision status, upgrade policy, and affected parent items in ADR(s); update proposed documents only after approval.

Out of scope:
- Installing tools, implementing build tasks, or choosing a database, cloud, scheduler, broker, or deployment product.

Affected modules and contracts:
- Repository/build boundary, local workflow, schema/OpenAPI toolchain.

Acceptance criteria:
- [ ] Given the proposed baseline and alternatives, when maintainers approve or defer each choice, then ADRs record the decision/status, owner, rationale, consequences, and affected backlog IDs.
- [ ] Given any choice deferred, when a dependent child is reviewed, then it remains blocked and no implementation assumes the deferred choice.

TDD first:
- First failing test: add a documentation-validation fixture requiring each decision record to contain Context, Decision/status, Consequences, Alternatives, Owner, and affected items; demonstrate failure against an incomplete draft. Code tests are N/A because this is an architecture decision.
- Additional coverage: validate Markdown links, ADR status/required headings, and references from BL-001 and BL-002.

Implementation notes:
- This child resolves only the language/build baseline; a recorded “proposed” choice is not acceptance.

Documentation updates:
- Functional: N/A; no user behavior changes.
- Installation: record intended local toolchain and supported targets only after approval.
- Operations/release: record build reproducibility and supported-platform implications.
- Architecture/ADR: create decision record(s); update software/technology documents with approved status and alternatives.

Security and data impact:
- No runtime data or secrets. Include supply-chain and tool-pin implications in the decision.

Compatibility, migration, and rollback:
- No runtime migration. Preserve a documented path to revise tool versions/layout without invalidating released contract identities.

Environment verification:
- Local: Markdown/ADR schema and links validate.
- Test: CI documentation check passes without installing an unapproved toolchain.
- Acceptance: record supported-target verification needed after the decision is implemented.
- Production promotion or smoke: N/A for a decision; record release implications, with no production change.

Dependencies:
- None.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus the decision evidence above.

### BL-001.2: Approve contract, record, and compatibility policy

Type: Architecture decision  
Status: Proposed  
Priority: P0

Problem: Ports, errors, metadata, records, and migration cannot be implemented compatibly until versioning and representations are explicit.

Scope:
- Decide contract/version compatibility, unsupported-version behavior, exact stable error envelope, canonical in-memory record boundary, metadata serialization policy, and adapter-owned transfer representation boundary.
- Specify migration/read/write compatibility, rollback limits, and decision gates for later persisted-state choices.

Out of scope:
- Choosing a database or binary/streaming data product; implementing schemas or ports.

Affected modules and contracts:
- Domain values, errors, capabilities, metadata and record contracts, adapter ports.

Acceptance criteria:
- [ ] Given supported and unsupported version fixtures, when the policy is applied, then each version is classified as compatible, migratable, or rejected with no partial side effect.
- [ ] Given a proposed record representation, when alternatives are reviewed, then ADRs explicitly preserve adapter-owned transfer formats and do not prescribe destination storage format.

TDD first:
- First failing test: a documentation/schema check rejects a decision record that omits compatibility direction, unsupported-version behavior, migration, or rollback. Code tests are N/A because this is a policy decision.
- Additional coverage: link and cross-reference checks for BL-003, BL-004, BL-006, BL-009, and BL-013.

Implementation notes:
- The architecture names JSON metadata as a proposed technology baseline; acceptance requires BL-001.1/this decision to confirm or revise it.

Documentation updates:
- Functional: document version guarantees and unsupported-version outcomes once approved.
- Installation: describe contract/schema distribution only after approval.
- Operations/release: specify compatibility windows and rollback constraints.
- Architecture/ADR: publish contract/compatibility decisions and update relevant Open Decisions.

Security and data impact:
- No runtime data; include unknown-field, untrusted-input, and sensitive-record handling consequences.

Compatibility, migration, and rollback:
- This decision defines the rules; it must not claim migration is reversible absent evidence.

Environment verification:
- Local: decision checklist and links validate.
- Test: version-fixture policy is specified for later hermetic execution.
- Acceptance: document compatibility/migration proof required before release.
- Production promotion or smoke: N/A now; decision must identify production compatibility gate.

Dependencies:
- None.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus the decision evidence above.

### BL-001.3: Approve authorization, activation, and governance policy

Type: Architecture decision  
Status: Proposed  
Priority: P0

Problem: Authorization inputs, approval authority, classification vocabulary, retention, and audit obligations remain open and affect metadata and run planning.

Scope:
- Decide authorization context and enforcement points, activation approval representation, data-classification vocabulary, retention/access policy ownership, and security/audit requirements.
- Record alternatives, owners, deferred choices, affected child IDs, and fail-closed expectations.

Out of scope:
- Selecting an identity provider, policy engine, secret store, audit backend, or implementing authorization.

Affected modules and contracts:
- Authoring, discovery, activation, planning, runtime scopes, secret references, audit and telemetry.

Acceptance criteria:
- [ ] Given authoring, activation, planning, and runtime actions, when the policy is reviewed, then each action has a named decision point and a specified deny behavior.
- [ ] Given a classification/retention choice that is deferred, when downstream items are refined, then they identify the unresolved policy and remain blocked where required.

TDD first:
- First failing test: a documentation validation check rejects an ADR missing enforcement points, denial semantics, data classes, retention ownership, or deferred gates. Code tests are N/A because this is a policy decision.
- Additional coverage: link check for BL-005, BL-007, BL-010, and BL-011 references.

Implementation notes:
- Explicitly distinguish approved baseline from Proposed recommendations. Do not prescribe permissive defaults.

Documentation updates:
- Functional: define user-visible permission and denial behavior after approval.
- Installation: list required identity/secret configuration only after choices are approved.
- Operations/release: state audit, retention, access, and incident-response implications.
- Architecture/ADR: decision record(s), alternatives, consequences, and source-document status.

Security and data impact:
- Decision is security-critical; assess least privilege, secret lifetime, classification, privacy, audit, retention, and deletion.

Compatibility, migration, and rollback:
- Document policy-version changes, active metadata impact, and rollback limits; no policy migration is implemented here.

Environment verification:
- Local: documentation/schema check and link validation.
- Test: specify denied-access and redaction test gates for BL-005.
- Acceptance: specify independently managed non-production identity/data policy checks.
- Production promotion or smoke: N/A now; record least-privilege smoke and audit evidence required.

Dependencies:
- None.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus security decision review and the evidence above.

### BL-001.4: Approve entrypoint and delivery-boundary decisions

Type: Architecture decision  
Status: Proposed  
Priority: P0

Problem: API/worker packaging, external request model, artifact identity and provenance, and deployment/rollback mechanisms are open.

Scope:
- Decide the initial API/worker binary boundary, request identity requirements, artifact/provenance/signing requirements, and which deployment decisions are release prerequisites versus later gates.
- Preserve the one-artifact test-to-acceptance-to-production promotion invariant.

Out of scope:
- Selecting or configuring a deployment platform, registry, orchestrator, or signing vendor; implementing public API or workers.

Affected modules and contracts:
- Entrypoints, request envelopes, artifact provenance, release and rollback gates.

Acceptance criteria:
- [ ] Given candidate entrypoint alternatives, when the decision is approved or deferred, then ADRs state boundaries, ownership, trade-offs, artifact identity, and dependent items.
- [ ] Given a proposed promotion, when the same-artifact rule is checked, then the policy prohibits rebuilding between test, acceptance, and production.

TDD first:
- First failing test: a documentation/configuration check fails when the ADR omits artifact identity, promotion order, rollback, or deferred product choices. Code tests are N/A because this is an architecture decision.
- Additional coverage: link/heading checks for BL-002, BL-014, BL-020 and the delivery baseline.

Implementation notes:
- Keep environment-specific products open unless separately approved.

Documentation updates:
- Functional: N/A until entrypoints exist.
- Installation: record entrypoint/package implications after approval.
- Operations/release: define artifact, promotion, rollback, and smoke evidence.
- Architecture/ADR: record boundary choices and update the status of the technology baseline.

Security and data impact:
- Include provenance, artifact integrity, credential isolation, and production approval implications.

Compatibility, migration, and rollback:
- No runtime migration. Specify how any later packaging change preserves contract identity and can restore the prior accepted artifact.

Environment verification:
- Local: ADR and links validate.
- Test: define provenance and exact-artifact checks for CI.
- Acceptance: define promotion proof against the candidate digest.
- Production promotion or smoke: no deployment here; specify restricted smoke and rollback evidence.

Dependencies:
- None.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus decision review and the evidence above.

## Parent coverage and blockers

Children .1–.4 cover BL-001's baseline/tooling, contract/version, security/governance, and entrypoint/delivery decisions. Checkpoint granularity, concurrency, audit implementation, and reference-adapter products are intentionally reserved for [BL-009](./BL-009-reference-profile.md); they remain open until that profile is approved. No child is Ready because no decision approval or ADR exists in the current repository.
