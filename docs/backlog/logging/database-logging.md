# Persist application logs in the database

## Desired outcome

Application log events can be inspected after they have been written to the configured database.

## Scope

Persist application log events using the shared levels and configuration defined in [Support distinct logging levels](logging-levels.md). This is diagnostic logging, not the audit trail for metadata edits.

## Acceptance criteria

- [x] Events at or above the configured threshold are persisted with a timestamp, level, source, and message.
- [x] Persisted entries remain available after the application restarts.
- [x] Events below the configured threshold are not persisted.
- [x] A failed database write is surfaced through the application's existing error-reporting path; it is not reported as a successful database log write.
- [x] Authentication credentials and API parameter values are not disclosed in persisted log entries.
- [x] A focused test verifies persistence, level filtering, restart durability, and exclusion of sensitive values.

## Dependencies

- [Support distinct logging levels](logging-levels.md)

## Implementation evidence

The [Liquibase changeset](../../../backend/src/main/resources/db/changelog/004-application-logging-and-metadata-audit.xml) creates durable application log storage; [LoggingService](../../../backend/src/main/java/io/mimir/logging/LoggingService.java) applies the shared threshold and surfaces failed inserts. [LoggingServiceTest](../../../backend/src/test/java/io/mimir/logging/LoggingServiceTest.java) covers filtering and write failures. [DockerStackTest](../../../tests/docker/test_stack.py) verifies persisted application-log rows survive a backend restart and checks that sensitive values are absent. `make test` passed.

## Implementation decisions

- General application logs are inspected with a database client; this iteration adds no log-search API or UI.
- Application log entries are retained indefinitely.
