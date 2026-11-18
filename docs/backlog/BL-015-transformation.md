# BL-015 refinement — Deterministic versioned transformation core

Parent: [BL-015](./backlog.md#bl-015-deliver-normalization-and-versioned-transformation)  
Status: In Progress  
Decision: [ADR-0003](../architecture/decisions/ADR-0003-deterministic-rules-and-replay.md)

## Outcome

Implement a small finite Go rule catalog with ordered, versioned operations,
input immutability, deterministic output identity, and provenance. The first
catalog is trim, case conversion, rename, integer conversion, and explicit
null replacement. No arbitrary language or transformation is inferred.

## Scope and evidence

Affected: `internal/transform`, JSON-object record contract, rule version
references. TDD first: `TestApplyUsesPinnedOrderedRulesWithoutMutatingInput`
was added before implementation and initially failed because `Rule`/`Apply`
were absent. Current coverage also rejects unknown, unversioned, and `v2`
rules; schema validation restricts the current catalog to `v1`.

Acceptance:
- [x] Ordered supported rules return deterministic output and ordered rule
  provenance without changing input.
- [x] Unsupported operations, missing fields, invalid conversions, duplicate
  identities, and missing versions fail explicitly.
- [x] Integrate the `v1` rule catalog with immutable run plans and execution,
  and reject unsupported rule versions in metadata, schema, and engines.

Compatibility/data: rule IDs and versions are pinned; unsupported versions are
rejected. No persisted data is migrated. Original records remain separate.
Rollback is binary rollback; no transformation outputs are rewritten.

Docs/environments: functional operation limits are in
[`docs/user/local-reference-ingestion.md`](../user/local-reference-ingestion.md).
Hermetic unit and local end-to-end tests run locally/CI; deployed acceptance
and production transformation verification are not applicable until an
environment is provisioned.
Security: no platform clients, secret resolution, or raw-value provenance logs.
Definition of Done remains subject to full workflow and independent review.
