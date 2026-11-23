# Definition of Done

This is the initial proposed quality bar for Mimir work. It applies to product changes and their directly related documentation; use the relevant criteria for documentation-only, maintenance, and other work. The Definition of Done is a local team agreement, not a checklist mandated by Scrum.

Work is Done when all applicable criteria are met:

- The agreed outcome and acceptance criteria are satisfied, and the change preserves intended existing behavior.
- Relevant automated tests and checks have been updated with the new functionality and pass. For application changes, this includes the applicable backend, frontend, and/or Docker integration tests. The full repository suite is available through `make test`; CI also runs backend and frontend verification and the complete Docker test target on pull requests.
- New or changed behavior has focused tests where appropriate, including important failure cases.
- Directly related user, technical, or process documentation is current.
- Known limitations, remaining work, and material risks are recorded in the appropriate backlog item or communicated with the change.
- For backlog-tracked work, the relevant backlog item is marked Done after all applicable completion criteria are met; any unfinished follow-up is tracked separately.
- When a pull request is used, its verification results are available and review feedback is addressed or explicitly resolved.
- The new artifact is running in the production environment. If a production environment has not been set up yet, the new artifact is running on the local machine.

Do not claim work is Done while a required test is failing, an acceptance criterion is unmet, or a known piece of required work is still outstanding. If a complete local suite cannot be run, record what was run and why; this does not replace required CI results.

## Current scope and evolution

The repository currently provides Java verification, Angular tests and a production build, test-runner checks, and container integration tests. Use `make test` for the complete local suite; see [technical setup](../technical/setup.md) for prerequisites and details.

The backlog says the Definition of Done should improve as features are delivered. Logging is a future backlog item and is **not** a default Done criterion yet. Add such a criterion only when the relevant feature has established and delivered that practice.
