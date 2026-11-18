# BL-001 refinement — Approve foundation decisions and ADR set

Parent: [BL-001](./backlog.md#bl-001-approve-foundation-decisions-and-adr-set)  
Workflow: [`docs/implementation/change-workflow.md`](../implementation/change-workflow.md)

## Outcome and current evidence

**Current:** The repository now has a local Go implementation, contracts, tests, CI, and ADR-0001 through ADR-0004. The system architecture is Accepted. The user explicitly delegated the open implementation choices; the ADRs record the conservative decisions and their limits. Independent maintainer review remains required before promotion.

**Required:** Decision records state context, decision, consequences, alternatives, status, owner, and implications; no dependent implementation silently chooses a platform product. The children below record delegated decisions as accepted for implementation, not as maintainer-reviewed production approval.

Decision authority: the user explicitly delegated these choices to the implementer for this delivery. Independent review owner remains the Mimir maintainers; no target milestone is assigned.

Blocker tracking: local implementation may proceed under the delegation. Items requiring hosted identity, external platforms, or production environments remain blocked with an owner and next action in the canonical backlog.

## Child work items

### BL-001.1: Approve the language, package, and local build baseline

Type: Architecture decision  
Status: In Progress (delegated decision recorded; independent review pending)
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
- [x] Given the proposed baseline and alternatives, when the delegated decision is made, then ADRs record decision status, authority, rationale, consequences, and affected backlog IDs.
- [x] Given any choice deferred, when dependent work is considered, then ADRs identify the later gate and implementation does not assume an unselected infrastructure product.

TDD first:
- First failing test: add a documentation-validation fixture requiring each decision record to contain Context, Decision/status, Consequences, Alternatives, Owner, and affected items; demonstrate failure against an incomplete draft. Code tests are N/A because this is an architecture decision.
- Additional coverage: validate Markdown links, ADR status/required headings, and references from BL-001 and BL-002.

Implementation notes:
- ADR-0001 selects Go 1.27.3, the supported build matrix, JSON Schema v1, standard-library runtime, Python documentation checks, and Make task interface. ADR-0002 selects a local file profile. These choices are accepted for this implementation by delegated authority and remain subject to independent review.

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
Status: In Progress (delegated decision recorded; independent review pending)
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
- [x] Given supported and unsupported version fixtures, when the policy is applied, then initial v1 is accepted and unsupported versions are rejected without partial side effects; no implicit migration is claimed.
- [x] Given the selected record boundary, when alternatives are recorded, then ADRs preserve adapter-owned transfer formats and do not prescribe destination storage format.

TDD first:
- First failing test: a documentation/schema check rejects a decision record that omits compatibility direction, unsupported-version behavior, migration, or rollback. Code tests are N/A because this is a policy decision.
- Additional coverage: link and cross-reference checks for BL-003, BL-004, BL-006, BL-009, and BL-013.

Implementation notes:
- ADR-0001 selects strict JSON metadata v1 and in-process JSON-object records. Unsupported versions and unknown fields fail closed. There is no prior persisted version to migrate.

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
Status: In Progress (delegated decision recorded; independent review pending)
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
- [x] Given authoring, activation, planning, and runtime actions, when the policy is applied locally, then each action has an explicit local-user decision point and fail-closed denial behavior.
- [x] Given hosted classification/retention policy that is deferred, downstream items identify it as a production gate and do not claim a hosted control.

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
Status: In Progress (delegated decision recorded; independent review pending)
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
- [x] Given candidate entrypoint alternatives, when the delegated decision is made, then ADRs state the local CLI boundary, deferred API/worker scope, artifact identity, and deployment consequences.
- [x] Given a proposed promotion, when the same-artifact rule is checked, then the policy requires the identical candidate for all environments and gates deployment on protected approvals.

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

Children .1–.4 cover BL-001's baseline/tooling, contract/version, security/governance, and entrypoint/delivery decisions. Checkpoint granularity and reference profile are resolved for the local implementation in ADR-0002; distributed concurrency, production audit implementation, hosted identity, signing, deployment products, and external adapters remain explicit later gates. The ADR set is present and validated, but the backlog remains In Progress pending independent maintainer review.
