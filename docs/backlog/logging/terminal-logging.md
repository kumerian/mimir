# Log to the terminal

## Desired outcome

Operators running Mimir locally can diagnose application activity and failures from the process terminal.

## Scope

Emit application log events to the terminal using the shared levels and configuration defined in [Support distinct logging levels](logging-levels.md). Do not include metadata audit history in this item.

## Acceptance criteria

- [x] Application startup and representative informational, warning, and error events appear in the terminal with a timestamp, level, source, and message.
- [x] Events below the configured logging threshold do not appear.
- [x] Log output does not disclose authentication credentials or API parameter values.
- [x] A focused test or automated check verifies the output format, level filtering, and exclusion of sensitive values.

## Dependencies

- [Support distinct logging levels](logging-levels.md)

## Implementation evidence

Startup and metadata events use the shared logger, and output is formatted with an ISO-8601 timestamp, level, source, and message. Messages exclude metadata values. See [ApplicationStartupLogger](../../../backend/src/main/java/io/mimir/logging/ApplicationStartupLogger.java), [LoggingService](../../../backend/src/main/java/io/mimir/logging/LoggingService.java), [MimirApplicationTest](../../../backend/src/test/java/io/mimir/MimirApplicationTest.java), and [LoggingServiceTest](../../../backend/src/test/java/io/mimir/logging/LoggingServiceTest.java). The API integration test checks that persisted log messages contain no sensitive test values; `make test` passed.
