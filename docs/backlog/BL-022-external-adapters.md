# BL-022 refinement — Defer external adapters pending evidenced need

Parent: [BL-022](./backlog.md#bl-022-add-independently-deployable-and-platform-specific-adapter-capabilities)  
Status: Blocked  
Decision: [ADR-0004](../architecture/decisions/ADR-0004-delivery-boundary-and-external-adapters.md)

## Decision and scope

No external database, cloud, scheduler, identity provider, API adapter host,
source, destination, or engine integration is selected. The local file profile
meets the evidence-backed reference slice; the repository contains no user
workload or operator requirement that would justify choosing a product. Do not
add integrations speculatively.

Acceptance:
- [x] The deferral and alternatives are recorded without changing core
  capability contracts or promoting the local profile to production.
- [ ] Resume implementation only when a concrete target and use case are
  documented with an accepted ADR, capability matrix, permissions, rollback,
  and test environment.

TDD first: documentation/ADR link validation is the current check; integration
tests are not applicable because no adapter is selected. No compatibility or
data migration is performed. Next action is an evidence-backed integration
request and operator-provisioned isolated test resources. This is a deliberate
scope deferral, not a claim that BL-022's integration acceptance criteria are
complete.
