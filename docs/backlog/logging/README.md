# Logging

Set up logging and display it in the UI. Thereafter, logging becomes a default part of the Definition of Done.

## First iteration

- [Support distinct logging levels](logging-levels.md)
- [Log to the terminal](terminal-logging.md) (depends on logging levels)
- [Persist application logs in the database](database-logging.md) (depends on logging levels)
- [Audit metadata changes made through the web interface](metadata-change-audit.md)

## Completion evidence

All four first-iteration items are complete and checked in the [backlog index](../backlog.md). The Definition of Done now includes shared, thresholded logging and sensitive-value protection. Evidence against the Done criteria:

- Outcome and behavior preservation: all four item acceptance checklists are checked; existing metadata CRUD and health behavior pass backend and Docker integration coverage.
- Automated and focused verification: `make test` passed (30 backend tests, 16 frontend tests, and 6 Docker integration tests); `npm run build` passed for the production frontend.
- Failure cases: tests cover filtered events, database logging failures, rejected metadata writes, audit redaction, and failed audit-history loading.
- Documentation and risks: [logging and metadata audit](../../technical/logging.md) documents configuration, SQL-only diagnostic log inspection, indefinite retention, redaction, and the unauthenticated local-development limitation.
- Completion and runtime: all four backlog items are checked; Docker integration tests started the three-container application, verified health, exercised the APIs, and verified log and audit persistence after a backend restart. No pull request was used.
