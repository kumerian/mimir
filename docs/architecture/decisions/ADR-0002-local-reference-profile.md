# ADR-0002: Local reference persistence and ingestion profile

## Status

Accepted for implementation under the user's explicit delegation; independent
maintainer review is still required before promotion. Local reference
implementations are not production-approved.

## Context

BL-009 left the reference source, destination, metadata/state/audit persistence,
checkpoint order, and recovery guarantees open. The initial product must work
from a clean checkout without a production account or external service.

## Decision

- Use local files and Go's standard library for the first reference profile.
  The profile uses separate metadata, execution-state, ingested-data, audit,
  and telemetry files under an explicitly configured local data directory.
  Metadata, run-state, audit, and discovery are local adapters behind their
  versioned ports and are composed by local use cases.
- Treat source JSON Lines as an immutable bounded snapshot identified by its
  content digest. There is no source acknowledgement operation.
- Treat a destination batch as committed only after its complete append-only
  journal record has been written, newline-terminated, flushed, and synced.
  Records are keyed by stable batch identity. Repeated identity with identical
  content is a no-op; identity with different content is a conflict.
- An incomplete final journal line is not a commit. On open, the adapter
  preserves all complete records and truncates only the unterminated tail.
  Complete malformed records fail closed and require operator recovery.
- The local run-state journal records destination commit evidence before its
  checkpoint. A checkpoint without durable commit evidence is rejected. Since
  the file source has no acknowledgement, resume reconciles the stable batch
  identity against the destination before doing any new commit.
- The run-state adapter conditionally appends lifecycle events under a local
  journal lock. Owner tokens and monotonically increasing attempts fence stale
  local workers; a retained state-directory lock requires an operator to
  verify process termination before cleanup. This is not distributed fencing.
- Append-only audit records are stored separately, deduplicated by event ID,
  and synced before a use case reports a required audit outcome as complete.
  If a later run stage is interrupted, rerunning unchanged inputs reconciles
  state and the audit journal; no cross-file atomic transaction is claimed.
  Structured local telemetry is a separate best-effort, redacted JSONL stream
  correlated by revision, plan, run, attempt, stage, and stable outcome.
  Metrics/traces exporters and remote telemetry are not implemented.
- The local profile supports one process at a time per data directory. It does
  not provide distributed locking, encryption-at-rest, backup automation,
  service-level guarantees, or production suitability. The operator owns
  directory permissions, backup, retention, and deletion.
- Startup verifies the built-in JSONL source and journal destination versions,
  batch limits, and bounded/idempotent/reconciliation guarantees without
  opening either adapter. Run configuration rejects a missing source root
  and non-private state directory. Health checks these static registrations,
  not filesystem, dependency, or environment readiness. Activation, discovery,
  and execution apply the local public/internal ceiling to effective object
  and column classifications, including previously activated revisions.
- Schedule evaluation persists stable requests separately from execution
  state and never dispatches them. Cancellation during evaluation and explicit
  local shutdown append versioned `stopped` acknowledgements to a separate
  request-ack journal, then ensure the corresponding redacted audit event.
  Stopping a request does not create or transition a run; stopped interval
  slots are not returned as pending by later evaluation.
- A local-only secret resolver port accepts only explicit `env://NAME`
  references in a synchronous callback with a five-second operation context
  and clears the temporary byte buffer on return. Callers must honor the
  context and not retain or copy the value. The current file
  source/destination adapters do not need or invoke it. Draft metadata may
  contain opaque references, but activation, discovery, and run reject them
  rather than silently ignoring a credential requirement. Previously active
  referenced revisions must be replaced with a new credential-free revision;
  no data is automatically migrated. This is not a production secret provider.
- Deterministic local fault injection exercises interruption before and after
  plan audit, attempt start, destination commit, quarantine commit, checkpoint,
  run audit, and terminal state persistence. Incomplete final journal lines
  exercise partial-append recovery. A source acknowledgement boundary is not
  applicable because the local file source explicitly lacks acknowledgement.
- The supported local boundary outcomes are:

  | Boundary | Durable outcome | Recovery |
  | --- | --- | --- |
  | Before source read completes | No run plan, destination commit, or checkpoint | Retry only with a validated active revision and a readable bounded snapshot |
  | Plan audit and attempt start | Immutable plan/audit and possibly a running attempt | Reuse the same plan/run identity; after process loss, verify and clear the local ownership lock before resuming under a new attempt |
  | Before destination commit | No destination batch commit | Retry the same stable batch identity |
  | After destination commit, before quarantine/checkpoint | Destination commit may exist | Reconcile by batch identity; never append blindly |
  | After quarantine, before checkpoint | Destination and quarantine may exist | Verify both identities and content, then persist the checkpoint |
  | After checkpoint, before required run audit | Destination and checkpoint exist, required audit may be absent | Rerun to reconcile; do not report success until audit is durable |
  | After run audit, before terminal state | Commit and audit exist, terminal state may be absent | Preserve the prior attempt and retry idempotently; audit is attempt-correlated |
  | Source acknowledgement | Not supported by the local file source | Not applicable; no acknowledgement is claimed |

- The first supported recovery claim is idempotent bounded-batch replay from
  an unchanged source snapshot to the local journal destination. It is not a
  claim of cross-system exactly-once delivery.

## Alternatives considered

- An in-memory implementation: simple but cannot demonstrate process restart
  recovery.
- A database or cloud store: unnecessary for the local reference profile and
  would introduce a product dependency without a deployment requirement.
- One combined journal for data/state/audit: conflates authorities and failure
  policies contrary to the accepted architecture.
- Source acknowledgement: the immutable file source has no acknowledgement
  protocol; inventing one would misstate its capability.
- Exactly-once across independent files: unsupported without atomic
  transactions; the profile instead uses destination deduplication and
  checkpoint-after-commit ordering.

## Consequences

Restart and duplicate scenarios can be exercised locally without services.
The profile is single-process and local-disk only. A durable filesystem is
required; network filesystems and concurrent processes are unsupported.
Complete malformed journal entries are not auto-repaired. An operator must
preserve the directory and use documented recovery procedures. Any production
adapter requires a separate ADR, contract tests, security review, backup and
restore evidence, and environment acceptance.

## Owner and affected backlog

Decision authority: explicit user delegation for this implementation; pending
independent maintainer review. Affects BL-004, BL-007, BL-009, BL-010, BL-011,
BL-012, BL-013, BL-014, and BL-018.
