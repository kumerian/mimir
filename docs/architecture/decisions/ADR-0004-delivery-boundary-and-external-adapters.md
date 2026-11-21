# ADR-0004: Candidate artifact and external adapter deferral

## Status

Accepted for implementation under the user's explicit delegation; repository
administrator and environment-operator configuration is still required.

## Context

BL-019–BL-022 cover entrypoints, promotion, operational documentation, and
independent adapters. The repository has no configured test/acceptance/
production environment, registry, signing identity, external adapter request,
or deployment target. The explicit user instruction prohibits direct
production deployment and PR creation.

## Decision

- Provide one local CLI binary with health, metadata validation/activation, and
  bounded run commands. Do not invent worker contracts or a general-purpose
  remote API in the absence of a deployment requirement. The later, concrete
  local Angular consumer and its narrow same-origin loopback HTTP boundary are
  recorded in [ADR-0005](./ADR-0005-local-gui-and-http-endpoint-boundary.md);
  [ADR-0006](./ADR-0006-local-compose-and-metadata-persistence.md) adds an
  explicit local container profile. Neither changes the CLI or authorizes
  hosted service behavior.
- GitHub Actions builds an identifiable candidate artifact only after
  validation jobs pass. The artifact includes a source revision, checksum, and
  contract identity. It is not signed and is not promoted by this workflow.
- Do not deploy or promote from this implementation run. Test deployment,
  acceptance promotion, production approval, exact-artifact promotion, and
  rollback workflows require a configured target and human approval.
- Do not add a platform-specific or separately deployed adapter for BL-022.
  The local file profile is sufficient to prove the bounded core use case and
  no integration need was evidenced at that time. ADR-0006 subsequently
  selects a local MariaDB metadata adapter for a concrete container profile.
  Other future adapters require a focused ADR and concrete need.
- Use GitHub Actions for repository CI only. GitHub Environments may be
  configured by repository administrators when target deployment workflows
  are defined; no assumptions are made about required reviewers or secrets.

## Alternatives considered

- Select a cloud, database, registry, signer, or deployment tool now: no
  evidence exists for a workload, operator, or hosted target, so such a choice
  would exceed the reference-slice need.
- Add generic remote HTTP/API and worker contracts without a deployment
  consumer: would freeze unsupported protocol and authorization semantics.
- Publish unsigned or unverified production releases: violates the
  same-candidate and human approval requirements.
- Use the file reference profile as a production store: its locking,
  durability, encryption, backup, and retention are insufficient.

## Consequences

Local source, tests, build, and candidate artifact are available. No external
deployment occurs and no production readiness is claimed. BL-019 is complete
only for the local CLI slice; ADR-0005's narrow local HTTP endpoint is now
implemented. The Angular connectivity shell and ADR-0006 local container
profile are implemented; full GUI journeys and production deployment are not.
BL-020 remains blocked
on operator-provisioned test/acceptance/production environments,
artifact-signing choice, approvals, and deployment target. BL-022 is
deliberately deferred for other integrations; the explicitly selected local
MariaDB metadata adapter is tracked under BL-023.

## Owner and affected backlog

Decision authority: explicit user delegation for this implementation; pending
independent review. Affects BL-019, BL-020, BL-021, and BL-022.
