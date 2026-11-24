# Support distinct logging levels

## Desired outcome

Application logs communicate the severity of an event, and operators can set a threshold to control which events are emitted.

## Scope

Define and implement the levels used by application logging, and make level filtering consistent across the configured log destinations. This item establishes levels for the terminal and database logging items; it does not add metadata change auditing.

## Acceptance criteria

- [x] The supported levels and their severity ordering are documented. At minimum, the implementation distinguishes informational, warning, and error events; any additional levels are documented.
- [x] Application events can be assigned an appropriate level, and the configured threshold includes events at that level or higher while excluding less severe events.
- [x] The default threshold and how an operator changes it are documented.
- [x] Tests verify that events on both sides of a configured threshold are emitted or filtered as expected.

## Dependencies and implementation decisions

- No dependencies.
- The default threshold is `INFO`; operators set `LOG_LEVEL` to `DEBUG`, `INFO`, `WARN`, or `ERROR`.

## Implementation evidence

`LOG_LEVEL` defaults to `INFO`; `DEBUG`, `INFO`, `WARN`, and `ERROR` share one threshold across terminal and database logging. See [LoggingService](../../../backend/src/main/java/io/mimir/logging/LoggingService.java), [runtime configuration](../../../backend/src/main/resources/application.yml), [Compose configuration](../../../compose.yaml), and [logging documentation](../../technical/logging.md). [LoggingServiceTest](../../../backend/src/test/java/io/mimir/logging/LoggingServiceTest.java) verifies threshold filtering and database failure propagation. `make test` passed.
