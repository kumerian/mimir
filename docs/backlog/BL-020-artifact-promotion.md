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
the identity manifest is derived from the candidate executable's health
contract list, with its source revision, working-tree status, binary digest,
and contract versions. CI checksums both the binary and identity manifest.
`make package` creates a local archive with the same manifest model. A local
archive inspection verified its embedded binary hash and complete contract
list; dirty working trees are explicitly ineligible as candidates. Automated
tests cover identity construction and invalid/unready executable responses.

Acceptance:
- [x] Passing CI builds an identifiable candidate artifact with source commit
  and SHA-256 identity.
- [ ] Promote the identical artifact to isolated test and acceptance targets,
  gate production on human approval, and verify rollback to the prior
  accepted artifact.
- [ ] Add signing/provenance, configuration validation, deployment smoke, and
  environment-specific recovery evidence.

TDD first: identity fixtures initially failed because the shared identity
module was absent; the scanner-pin regression test initially failed because
the local Make target used a different release than CI. Both are now retained
as passing tests. Deployment/promotion fixtures remain blocked because no
deployment target was guessed. Next action: repository operators configure
protected environments, signing, and a supported runtime target, then refine
the deployment workflow. No production action is authorized here.
