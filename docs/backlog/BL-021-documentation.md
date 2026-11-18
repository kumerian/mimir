# BL-021 refinement — Local functional and operating documentation

Parent: [BL-021](./backlog.md#bl-021-publish-complete-functional-installation-operations-and-acceptance-documentation)  
Status: In Progress  
Decision: [ADR-0001](../architecture/decisions/ADR-0001-foundation-and-contracts.md),
[ADR-0002](../architecture/decisions/ADR-0002-local-reference-profile.md)

## Outcome

Document only behavior that exists: local metadata validation/activation,
bounded ingestion, supported limits, recovery, state ownership, and explicit
non-production limitations.

Verification evidence: `make install-smoke` builds a binary into a temporary
install directory and executes health, validation/report lookup, activation,
and a synthetic run with temporary cleanup. It passed on Linux/amd64. CI
defines native install smoke on Linux, macOS, and Windows runners, while its
build matrix compiles the six declared target combinations; hosted execution
has not been observed in this session.
An offline synthetic whole-directory backup and restore drill
(`TestOfflineWholeDirectoryBackupRestorePreservesReplay`) verifies that the
restored local state reconciles the original run without a duplicate commit.
It is not a candidate promotion, migration, or other-host restore drill.

Acceptance:
- [x] README links to getting started and local operation.
- [x] Functional and operations guides include synthetic examples, permissions,
  failures, recovery, compatibility, cleanup, and non-implemented behavior.
- [ ] Verify clean installation on all supported developer targets, migration,
  backup/restore drills, and promotion/rollback procedures against a candidate.

TDD first: documentation verification is `make docs`; it validates local
Markdown links, ADR sections, JSON contract syntax/version, and executable
metadata validation. The first docs check failed on headings generated from
backlog anchors; the checker was corrected and now passes.

No runtime migration or deployment is introduced. User-data examples are
synthetic. Test and acceptance docs remain constrained by absent environments;
production runbooks are not claimed as operationally tested.
