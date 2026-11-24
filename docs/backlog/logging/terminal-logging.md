# Log to the terminal

## Desired outcome

Operators running Mimir locally can diagnose application activity and failures from the process terminal.

## Scope

Emit application log events to the terminal using the shared levels and configuration defined in [Support distinct logging levels](logging-levels.md). Do not include metadata audit history in this item.

## Acceptance criteria

- Application startup and representative informational, warning, and error events appear in the terminal with a timestamp, level, source, and message.
- Events below the configured logging threshold do not appear.
- Log output does not disclose authentication credentials or API parameter values.
- A focused test or automated check verifies the output format, level filtering, and exclusion of sensitive values.

## Dependencies

- [Support distinct logging levels](logging-levels.md)
