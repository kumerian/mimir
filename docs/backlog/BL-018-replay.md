# BL-018 refinement — Replay eligibility and pinned input checks

Parent: [BL-018](./backlog.md#bl-018-deliver-deterministic-replay-and-recovery-guarantees)  
Status: In Progress  
Decision: [ADR-0003](../architecture/decisions/ADR-0003-deterministic-rules-and-replay.md)

## Outcome

Prepare a new replay identity only when metadata revision/digest, source
snapshot, ordered adapter versions, and ordered rule versions match. Require a
bounded reason enum and preserve original identity.

## Scope and evidence

Affected: `internal/replay` plan contract. TDD first:
`TestPrepareReplayPinsOriginalInputsAndRejectsChangedSource` was added before
implementation and initially failed because `Prepare`/`Plan` were absent.

Acceptance:
- [x] Missing or changed source snapshot is rejected; equal pinned inputs
  produce a replay plan linked to the original and a distinct identity.
- [x] Connect replay preparation to persisted immutable plans, local CLI,
  cancellation, audit, and reconciliation of the original idempotent batch.
- [x] Verify fault matrix across supported commit/checkpoint boundaries. Run
  recovery covers before/after destination, quarantine, checkpoint, audit, and
  terminal-state boundaries; replay restart tests cover replay-plan write,
  preparation audit, and original-batch reconciliation boundaries.

Compatibility/data: original attempt remains immutable; unsupported plan
versions reject. The command prepares a distinct replay identity but reconciles
the original batch; it does not create a new output-producing execution.
Rollback cancels only a new replay preparation. Unit tests are hermetic;
acceptance/production replay requires an approved adapter profile and is not
available. No replay guarantee is claimed for arbitrary sources.
