# BL-020 refinement — Candidate artifact identity; promotion blocked

Parent: [BL-020](./backlog.md#bl-020-establish-immutable-artifact-packaging-promotion-and-rollback)  
Status: Blocked  
Decision: [ADR-0004](../architecture/decisions/ADR-0004-delivery-boundary-and-external-adapters.md)

## Outcome and current state

Build a traceable candidate with source revision and binary checksum. Exact
test/acceptance/production promotion, artifact signing, runtime deployment, and
rollback require a provisioned deployment target, human approval policy, and
signing/provenance configuration. None exists in the checkout.

## Scope and evidence

Candidate artifact construction and identity are in `.github/workflows/ci.yml`;
`make package` creates a local archive with a manifest/checksum. The local
package test has been exercised.

Acceptance:
- [x] Passing CI builds an identifiable candidate artifact with source commit
  and SHA-256 identity.
- [ ] Promote the identical artifact to isolated test and acceptance targets,
  gate production on human approval, and verify rollback to the prior
  accepted artifact.
- [ ] Add signing/provenance, configuration validation, deployment smoke, and
  environment-specific recovery evidence.

TDD first: a workflow fixture verifying artifact identity/promotion has not yet
been implemented; no deployment target was guessed. Next action: repository
operators configure protected environments, signing, and a supported runtime
target, then refine the deployment workflow. No production action is
authorized here.
