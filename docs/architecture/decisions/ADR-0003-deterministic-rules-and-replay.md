# ADR-0003: Deterministic rule execution and bounded replay policy

## Status

Accepted for implementation under the user's explicit delegation; independent
maintainer and security review remains required.

## Context

BL-015–BL-018 need rule behavior without selecting a transformation language,
statistical engine, external product, or unsafe replay guarantee. The accepted
architecture requires quality evaluation to remain separate from cleaning and
requires replay to pin all relevant inputs.

## Decision

- Implement rule functions in Go using a finite, versioned operation catalog.
  No arbitrary code, expressions, scripts, or external transformation language
  is accepted in metadata.
- Transformation rules execute in declared order and require stable rule ID
  and version. The initial operation subset is string trim/case conversion,
  field rename, signed integer conversion, and explicit null replacement.
  The only currently supported rule version is `v1`; all other versions fail
  closed in metadata validation and runtime engines. Inputs are copied;
  provenance records rule identity and operation.
- Quality evaluation is pure and supports explicit required/type/numeric
  bound/enum checks. The caller must select a failure disposition from warning,
  reject, quarantine, hold, or cleaning-required. Unknown operations fail
  explicitly. The only currently supported quality rule version is `v1`.
  Quality never invokes cleaning.
- Cleaning is a separate explicit call requiring an actor and authorization
  result. Initial correction operations are trim/case conversion and
  null-replacement with an explicit replacement value. The original value is
  identified by digest; the cleaned output is distinct. `v1` is the only
  supported correction version. The local CLI persists output separately and
  writes a redacted audit event; cleaning is not invoked by ingestion or
  quality disposition.
- Replay preparation is allowed only when metadata revision/digest, source
  snapshot digest, ordered adapter versions, and ordered rule versions match.
  Reasons are fixed values (`operator_request`, `recovery`, `incident`) to avoid
  arbitrary sensitive text in replay metadata. Missing inputs reject replay;
  no fallback to current metadata is allowed.
- The local reference run uses a stable source snapshot and idempotent
  destination batch identity. It reconciles the destination before retrying,
  then persists a checkpoint. This is a bounded local replay guarantee, not
  distributed exactly-once behavior.

## Alternatives considered

- User-provided executable transformations: flexible but unsafe, nondeterministic,
  and not reviewable as a core contract.
- Implicit type coercion/cleaning: risks silent data changes and loss of
  provenance.
- Replaying against current metadata or current source: produces a different
  result while implying equivalence.
- Free-form replay reason strings: useful operationally but can carry secret
  or sensitive payload text into audit/state.

## Consequences

The rule catalog is intentionally small; additional operations or versions
require compatibility fixtures, versioned tests, schema updates, and
documentation. Transformation and quality are integrated into the local
ingestion stages; reject and quarantine are durable and reconciled before the
checkpoint. Cleaning is explicit and separately persisted/audited. Replay
preparation is exposed by the CLI and reconciles the original pinned batch;
it does not create a new output-producing run. Production adapters,
distributed recovery, and a complete crash-boundary matrix remain
unimplemented.

## Owner and affected backlog

Decision authority: explicit user delegation for this implementation; pending
independent review. Affects BL-015, BL-016, BL-017, and BL-018.
