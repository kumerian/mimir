# BL-016 refinement — Pure quality evaluation and explicit disposition

Parent: [BL-016](./backlog.md#bl-016-deliver-quality-evaluation-quarantine-and-disposition)  
Status: In Progress  
Decision: [ADR-0003](../architecture/decisions/ADR-0003-deterministic-rules-and-replay.md)

## Outcome

Implement pure, versioned required/type/min/max/enum checks. Quality reports
failures and a caller-selected disposition; it never mutates records or calls
cleaning.

## Scope and evidence

Affected: `internal/quality`, record and rule contracts. TDD first:
`TestEvaluateReportsDispositionWithoutMutatingRecord` was added before
implementation and initially failed because `Evaluate`/`Rule` were absent.

Acceptance:
- [x] Failures are correlated to rule ID/version/field and change only
  disposition evidence; input is unchanged.
- [x] Missing disposition and unknown operations fail explicitly.
- [x] Integrate `v1` quality rules, warning/reject/quarantine publication,
  separate durable quarantine, local classification ceiling, audit, and
  checkpoint recovery; unsupported dispositions fail closed.

Compatibility/data: quality rule versions are explicit; unsupported versions
fail closed. Quarantine is retained in the operator-owned state directory
until the operator removes that complete directory; there is no implicit
expiry or row-level purge policy. Rollback restores the prior binary and
leaves source records unchanged.

Docs/environments: local capabilities and non-integration limits are described
in the user guide. Hermetic unit tests run locally/CI. Acceptance quarantine
and production policy tests are blocked until a destination/policy profile is
approved. Security: no raw value in failure diagnostics or audit.
