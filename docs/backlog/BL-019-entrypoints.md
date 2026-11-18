# BL-019 refinement — Local CLI and process identity

Parent: [BL-019](./backlog.md#bl-019-deliver-cli-api-worker-and-composition-entrypoints)  
Status: In Progress  
Decision: [ADR-0001](../architecture/decisions/ADR-0001-foundation-and-contracts.md),
[ADR-0004](../architecture/decisions/ADR-0004-delivery-boundary-and-external-adapters.md)

## Outcome

Deliver one thin local CLI for health, strict metadata validation, explicit
metadata activation, and a bounded run. CLI translation must not perform
business transitions itself.

Acceptance:
- [x] Health reports build/contract/profile identity without secrets.
- [x] Validation and activation expose stable failures; local run requires an
  explicit local grant and emits no record values.
- [x] Active revision and plan/capability identity are recorded by use cases.
- [ ] Implement authentication for hosted callers, OpenAPI/API, worker
  ownership, contract-test host, and independently reviewed command contract.

TDD first: `TestHealthReportsBuildAndContractIdentity`,
`TestRunRequiresExplicitLocalGrant`, and
`TestLocalActivateAndRunCommandsDoNotEmitRecordValues` cover observed CLI
behavior. Compatibility: configuration and JSON contract versions reject
unknown forms; no persisted migration. Security: local profile only; no
production identity/secret provider.

Local tests/build/docs are verified. No hosted test or acceptance deployment
is configured; production smoke is prohibited from this implementation run.
