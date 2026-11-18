# BL-006 refinement — Establish versioned metadata contracts and schema validation

Parent: [BL-006](./backlog.md#bl-006-establish-versioned-metadata-contracts-and-schema-validation)  
Workflow: [`docs/implementation/change-workflow.md`](../implementation/change-workflow.md)

## Outcome and current evidence

**Current:** Strict metadata JSON v1 has an executable Draft 2020-12 validator and parser, semantic checks, duplicate/unknown/version rejection, deterministic content-free validation reports, rule-version fixtures, and a synthetic draft example. The exact schema is embedded and freshness-checked. Drafts and reports are not runnable. No previous persisted version exists and migration is not claimed.

Children deliver the schema/domain contract and deterministic parsing/validation compatibility behavior. Persistence and activation are BL-007.

Owner / target milestone for all children: **Open** — no owner or milestone is recorded in the repository; assign before moving any child to Ready.

Blocker tracking: For each **Blocked** child, its unresolved dependencies below are the blocker; the next action is to complete the first unresolved prerequisite in that list. Blocker owner is **Open** and must be assigned by maintainers.

## Child work items

### BL-006.1: Define versioned metadata model and schema surface

Type: Technical enabler  
Status: In Progress
Priority: P0

Problem: Metadata authoring cannot be validated until technical, functional, scheduling, governance, and rule data have an explicit versioned model.

Scope:
- Define source/object/column/mapping/schedule/rule/revision structures, inheritance points, provenance, secret references and classification fields.
- Publish versioned contract/schema artifacts after BL-001.2 approves serialization and BL-001.3 settles governance fields.

Out of scope:
- Repository storage, discovery, activation, a full rule catalog, or a chosen persistence product.

Affected modules and contracts:
- Metadata domain aggregate, JSON/YAML authoring boundary if approved, schemas and examples.

Acceptance criteria:
- [x] Given a valid synthetic metadata document for each required entity, when the approved schema validates it, then it yields a versioned candidate with no execution authority.
- [x] Given an unknown field, unsupported version, or invalid reference, when metadata parsing/validation runs, then it returns a specific safe field/version diagnostic and does not create an active revision. Unknown field names that resemble secret material are redacted; parsing is side-effect free.

TDD first:
- First failing test: add a fixture with an unknown field and assert validation rejects it without producing an executable revision.
- Additional coverage: valid/invalid entities, required fields, inheritance, classification, secret-reference shape, and schema version.

Implementation notes:
- JSON/YAML/JSON Schema are Proposed in technology baseline; use only if BL-001 accepts them. Do not invent canonical record serialization here.

Documentation updates:
- Functional: metadata field/schema reference and synthetic examples.
- Installation: how schema/version assets and validator are obtained and used.
- Operations/release: unsupported-version and metadata compatibility diagnostics.
- Architecture/ADR: reflect the approved contract and serialization decision.

Security and data impact:
- Reject embedded secret values; include data-classification and access policy fields only as approved.

Compatibility, migration, and rollback:
- Include previous/current/unsupported schema fixtures and migration compatibility contract; no destructive migration.

Environment verification:
- Local: hermetic schema validation and examples.
- Test: schema fixture and generated-artifact freshness checks in CI.
- Acceptance: validate candidate metadata and prior-version fixtures against the exact artifact.
- Production promotion or smoke: no activation here; production compatibility gate is documented.

Dependencies:
- BL-001.2.
- BL-001.3.
- BL-003.1.
- BL-005.1.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus schema and compatibility evidence above.

### BL-006.2: Parse authoring input and produce deterministic validation reports

Type: Technical enabler  
Status: In Progress
Priority: P0

Problem: Authoring inputs require deterministic structural, semantic, compatibility, and security validation before they can be used by a use case.

Scope:
- Implement canonical parsing for approved authoring formats, duplicate-key detection, unknown-field policy, deterministic diagnostics, and version checks.
- Represent validation outcomes with field/rule/scope, security impact and compatibility decision; preserve source provenance.

Out of scope:
- Activating metadata, automatically repairing input, or modifying active metadata.

Affected modules and contracts:
- Metadata parser/validator, validation report, schema fixtures and diagnostics.

Acceptance criteria:
- [x] Given the same valid input, when parsed repeatedly, then its canonical form and validation report are deterministic.
- [x] Given duplicate keys, unknown fields, unsupported versions, or semantic violations, when parsing/validation runs, then it rejects them according to the strict v1 policy without partial side effects. Fixtures cover duplicate/unknown/version rejection and invalid opaque references.
- [x] Given invalid metadata, when a caller asks for execution authority, then no run-eligible revision is returned.

TDD first:
- First failing test: provide duplicate keys whose values disagree and assert parser rejects the document rather than silently selecting one.
- Additional coverage: valid/invalid/unknown/version fixtures, semantic boundaries, stable diagnostics, migration compatibility and security failures.

Implementation notes:
- Do not silently normalize invalid content or infer cleaning/transformation from descriptive metadata.

Documentation updates:
- Functional: authoring, validation diagnostics, limits and examples.
- Installation: validator invocation, schema/tool dependencies, and troubleshooting.
- Operations/release: compatibility rejection and metadata rollback guidance.
- Architecture/ADR: ensure validation/activation remains separate.

Security and data impact:
- Treat authoring input as untrusted; enforce size/depth limits as approved, avoid logging payloads/secrets, and classify diagnostics.

Compatibility, migration, and rollback:
- Test old supported input and unsupported-version rejection. Migration must preserve provenance and be non-destructive; rollback means retain prior candidate.

Environment verification:
- Local: synthetic parser/schema suite.
- Test: hermetic version, unknown-field and duplicate-key tests.
- Acceptance: validate representative non-production metadata using the candidate artifact.
- Production promotion or smoke: no production metadata writes; verify preflight compatibility before future deployment.

Dependencies:
- BL-006.1.
- BL-004.1.
- BL-003.1.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus parser/validation evidence above.

## Parent coverage and blockers

Children cover all parent metadata entities, versioned schema, YAML/JSON only if approved, duplicate/unknown/version handling, deterministic reports and compatibility fixtures. Persistence, revision history and activation remain BL-007. Execution stays blocked on BL-001.2/.3.
