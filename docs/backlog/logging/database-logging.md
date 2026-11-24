# Persist application logs in the database

## Desired outcome

Application log events can be inspected after they have been written to the configured database.

## Scope

Persist application log events using the shared levels and configuration defined in [Support distinct logging levels](logging-levels.md). This is diagnostic logging, not the audit trail for metadata edits.

## Acceptance criteria

- Events at or above the configured threshold are persisted with a timestamp, level, source, and message.
- Persisted entries remain available after the application restarts.
- Events below the configured threshold are not persisted.
- A failed database write is surfaced through the application's existing error-reporting path; it is not reported as a successful database log write.
- Authentication credentials and API parameter values are not disclosed in persisted log entries.
- A focused test verifies persistence, level filtering, restart durability, and exclusion of sensitive values.

## Dependencies

- [Support distinct logging levels](logging-levels.md)

## Open questions

- The backlog does not specify how operators will search or view general application log entries, or how long they are retained. Decide these before implementation if either is required for the first iteration.
