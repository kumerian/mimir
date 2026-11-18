# BL-017 refinement — Explicit auditable cleaning helper

Parent: [BL-017](./backlog.md#bl-017-deliver-explicit-deterministic-cleaning-with-complete-provenance)  
Status: In Progress  
Decision: [ADR-0003](../architecture/decisions/ADR-0003-deterministic-rules-and-replay.md)

## Outcome

Provide a separate explicit cleaning call that requires actor and authorization,
applies a versioned correction to a copy, retains an input digest, and emits
redacted audit fields. It is not invoked by quality evaluation.

## Scope and evidence

Affected: `internal/cleaning`, cleaning rule/result contracts. TDD first:
`TestApplyIsExplicitAuditableAndDoesNotReplaceInput` was added before
implementation and initially failed because `Apply`/`Correction` were absent.

Acceptance:
- [x] Supported corrections preserve input and return an output digest,
  correction version, actor, input reference, and deterministic idempotency
  key.
- [x] Unauthorized, unsupported, or invalid corrections fail closed.
- [x] Integrate local explicit authorization and classification ceiling,
  preview, separate idempotent output persistence, redacted durable audit, and
  recovery when output is committed before audit.

Compatibility/data: original record is never overwritten; no persisted
migration. Only correction version `v1` is supported. Rollback is binary
rollback with both representations retained.
Hermetic tests run locally/CI; acceptance/production execution is blocked
because no controlled-data/cleaning deployment profile exists. Review the
security boundary before enabling runtime use.
