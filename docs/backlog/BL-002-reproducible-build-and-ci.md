# BL-002 refinement — Establish reproducible build, local workflow, and core CI

Parent: [BL-002](./backlog.md#bl-002-establish-reproducible-go-build-local-workflow-and-core-ci)  
Workflow: [`docs/implementation/change-workflow.md`](../implementation/change-workflow.md)

## Outcome and current evidence

**Current:** A pinned Go module, Make tasks, repository checks, GitHub Actions matrix, candidate identity, package script, and functional setup guide exist. `make test`, `make lint`, `make docs`, all six cross-builds, and the Linux/amd64 clean-install smoke pass locally. Local and CI `govulncheck` now share the same pinned version; a regression test enforces that. Candidate identity is derived from the executable's health contract list, and synthetic tests verify binary hashing, revision/status fields, rejection of incomplete health, and dirty-tree ineligibility. The vulnerability database scan is blocked here by DNS; hosted CI and non-Linux clean-install execution remain unverified.

Children below cover the parent outcome: approved repository/toolchain/local tasks, then CI verification and candidate build evidence. Neither child implements product behavior or deployment promotion.

Owner / target milestone for all children: **Open** — no owner or milestone is recorded in the repository; assign before moving any child to Ready.

Blocker tracking: For each **Blocked** child, its unresolved dependencies below are the blocker; the next action is to complete the first unresolved prerequisite in that list. Blocker owner is **Open** and must be assigned by maintainers.

## Child work items

### BL-002.1: Establish the approved module and hermetic local workflow

Type: Technical enabler  
Status: In Progress
Priority: P0

Problem: Contributors lack a reproducible local path to build and validate changes.

Scope:
- Establish the repository/module layout, pinned tools, dependency policy, and version-controlled tasks from BL-001.1.
- Provide fast hermetic `test`, race/contract/integration classifications, format, lint, vet, dependency, build, and docs task surfaces.
- Add clean-checkout contributor setup and non-secret configuration.

Out of scope:
- Product use cases, external service dependencies in default tests, acceptance infrastructure, or release promotion.

Affected modules and contracts:
- Repository/build layout, task interfaces, test classification, generated-artifact checks.

Acceptance criteria:
- [x] Given a supported developer checkout, when documented tasks run, then format, lint, vet, module verification, docs checks, hermetic tests, and Linux build complete without production credentials. Clean-checkout install is not yet exercised on every target.
- [x] Given unavailable network services or credentials, when default tests run, then tests remain hermetic and do not depend on ambient services.

TDD first:
- First failing test: add a clean-checkout task smoke test asserting every required task exists and default test configuration has no external-service dependency; run it before adding tasks.
- Additional coverage: pin-integrity, module-tidy, generated-file freshness, platform build, and task failure-propagation checks.

Implementation notes:
- Follow BL-001.1 decisions; do not make a local helper a hidden global prerequisite.

Documentation updates:
- Functional: N/A.
- Installation: contributor clean setup, supported OS/runtime, configuration, verification, cleanup, troubleshooting.
- Operations/release: document build outputs and local artifact limits.
- Architecture/ADR: update repository layout only if it differs from approved software architecture.

Security and data impact:
- No product data. Pin dependencies/tools and scan for secrets; use synthetic fixtures.

Compatibility, migration, and rollback:
- No persisted state. Document module/dependency changes; revert task/tool config atomically if validation is broken.

Environment verification:
- Local: clean checkout runs pinned tasks and hermetic suite.
- Test: same commands run in isolated CI without ambient credentials.
- Acceptance: no runtime acceptance deployment; verify packaged setup prerequisites are documented.
- Production promotion or smoke: N/A for tooling foundation; artifact requirements are captured for BL-002.2/BL-020.

Dependencies:
- BL-001.1.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus the task and clean-checkout evidence above.

### BL-002.2: Gate CI on reproducible checks and candidate build identity

Type: Technical enabler  
Status: In Progress
Priority: P0

Problem: A reviewed source revision needs visible pass/fail/not-applicable/blocked results and an identifiable build artifact.

Scope:
- Add PR/merge CI for pinned format, lint/vet, unit/contract/race, dependency/architecture, docs/schema, secret/vulnerability checks, and supported builds as approved.
- Report each affected check status explicitly; retain source revision and artifact digest/provenance for candidates.
- Document clean CI setup and diagnostic recovery.

Out of scope:
- Acceptance/production deployments, signing vendor or registry selection, and product runtime.

Affected modules and contracts:
- CI workflow, task exit codes, candidate build identity and provenance metadata.

Acceptance criteria:
- [x] Given a deliberately failing required check, when CI runs, then the named-check runner and its deliberate-failure test prove a nonzero result and identify the check; CI invokes that runner. Hosted execution remains unobserved here.
- [x] Given an unaffected check, when it is not run, then the local tagged-suite runner reports “not applicable,” not success-by-omission.
- [x] Given a passing source revision, when candidate artifacts are built, then each artifact records its source revision and immutable digest. CI candidate construction is configured; this run did not execute GitHub Actions.

TDD first:
- First failing test: workflow-validation test with fixtures for failed, passed, not-applicable, and blocked job states; it must reject a missing required status.
- Additional coverage: build reproducibility, dependency/vulnerability/secret checks, generated output freshness, and artifact metadata validation.

Implementation notes:
- Depends on the approved tool and artifact decisions; production artifact signing/deployment remains separately scoped.

Documentation updates:
- Functional: N/A.
- Installation: CI prerequisites and local reproduction commands.
- Operations/release: CI status meanings, candidate identity, failure diagnosis, and rollback boundary.
- Architecture/ADR: reference BL-001.4 decision; no undocumented deployment assumptions.

Security and data impact:
- Protect tokens and build logs; do not expose secrets or production data. Apply dependency/supply-chain checks.

Compatibility, migration, and rollback:
- CI configuration changes do not migrate runtime state. Pin workflow/tool changes and preserve traceability to source revision.

Environment verification:
- Local: task/workflow validators and clean build.
- Test: PR and merge CI exercise required status matrix against synthetic fixtures.
- Acceptance: no deployment; document the later exact-candidate promotion proof.
- Production promotion or smoke: N/A until BL-020; do not claim this CI change deployed an artifact.

Dependencies:
- BL-001.1.
- BL-001.4.
- BL-002.1.

Definition of Done:
- Apply `docs/implementation/change-workflow.md` in full, plus CI and candidate-identity evidence above.

## Parent coverage and blockers

Children cover reproducible local development, clean-checkout documentation, core CI, explicit status reporting, and candidate artifact identity. BL-001.1 and BL-001.4 block execution; exact approved tool/version/platform choices remain Open.
