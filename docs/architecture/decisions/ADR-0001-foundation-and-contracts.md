# ADR-0001: Foundation, metadata, and contract baseline

## Status

Accepted for implementation under the user's explicit delegation; independent
maintainer review is still required before promotion.

## Context

The accepted system architecture requires a platform-neutral core, validated
metadata, declared capabilities, immutable run inputs, safe recovery, and
separation of data, state, and audit. The repository had no implementation.
BL-001 left language, metadata encoding, compatibility, local authorization,
and process shape open. The user explicitly delegated those decisions to the
implementer for this delivery.

## Decision

- Implement the first release as one Go module and one process, using Go
  1.27.4. Support Linux, macOS, and Windows on amd64 and arm64 as build targets;
  CI must build all six combinations. Linux is the verified local target.
- Use the Go standard library for the initial core except for the pinned
  `github.com/santhosh-tekuri/jsonschema/v6` Draft 2020-12 validator required
  to enforce the published metadata contract at runtime. The validator and
  schema are embedded in the binary; the runtime does not fetch a schema or
  contact a registry. `make` is the task interface; Python 3.12's standard
  library performs repository/documentation checks. Tool versions are pinned
  in repository files or CI.
- Use versioned JSON metadata (`schemaVersion: "mimir.metadata/v1"`). YAML is
  not accepted in v1. Reject duplicate keys, unknown fields, malformed values,
  and unsupported versions. Do not silently migrate. No persisted previous
  format exists, so v1 is the initial supported version.
- Use typed Go values in-process. JSON object records are the initial canonical
  record boundary; keys are canonicalized deterministically. Adapter-owned
  storage/transfer encodings remain replaceable and are not prescribed by this
  choice.
- Represent stable failures by a versioned category, operation/stage, retry
  signal, commit certainty, cause reference, and reconciliation requirement.
  Cancellation and timeout are distinct categories. An unknown commit outcome
  is never safe for blind retry.
- Authorization is evaluated through an application-owned policy port. Missing
  or indeterminate grants deny access. The initial local development policy
  requires explicit local caller identity and grants; it is not an identity
  provider or production authorization system.
- The initial process exposes a local CLI and health/readiness information.
  API and worker binaries are not introduced until a concrete boundary is
  needed. Commands call application services; they do not own domain
  transitions.
- Classification labels are `public`, `internal`, `confidential`, and
  `restricted`. No retention duration is inferred; local reference data and
  journals are developer-owned and must be deleted explicitly.

## Alternatives considered

- Python: convenient for scripting but less aligned with the proposed Go
  module/test boundary and static executable delivery.
- Multi-module/multi-process first release: adds protocol and deployment
  complexity before contracts are proven.
- Permissive unknown-field handling or implicit migration: risks silently
  changing metadata meaning.
- A production identity provider or secret store: no environment or operational
  requirement exists to justify selecting one.
- A wire/binary record format: no selected adapter requires one; adapter
  implementations retain that choice behind ports.

## Consequences

The local developer path is deterministic and uses pinned module dependencies.
Metadata v1 is intentionally strict and can reject forward-added fields;
future compatibility must use a new reviewed contract policy. The architecture
remains modular inside one module. The Go version and support matrix are a
project-level decision and require CI verification; they do not imply
production suitability for local adapters. Production identity, retention
periods, and independently deployed protocols remain separate decisions.

The explicit validator dependency is a narrow exception to the standard
library baseline: it provides actual Draft 2020-12 semantics (including local
references and conditionals) rather than a partial hand-written schema
interpreter. Its version is pinned in `go.mod`/`go.sum`, and the embedded
schema must byte-match the canonical contract in CI. The validator's Go regexp
implementation does not support ECMA look-around, so the path constraint uses
portable schema patterns while domain semantic validation remains authoritative
for additional invariants.

## Owner and affected backlog

Decision authority: explicit user delegation for this implementation; pending
independent maintainer review. Affects BL-001, BL-002, BL-003, BL-005, BL-006,
BL-019, BL-020, and BL-021.
