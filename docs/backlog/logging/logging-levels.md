# Support distinct logging levels

## Desired outcome

Application logs communicate the severity of an event, and operators can set a threshold to control which events are emitted.

## Scope

Define and implement the levels used by application logging, and make level filtering consistent across the configured log destinations. This item establishes levels for the terminal and database logging items; it does not add metadata change auditing.

## Acceptance criteria

- The supported levels and their severity ordering are documented. At minimum, the implementation distinguishes informational, warning, and error events; any additional levels are documented.
- Application events can be assigned an appropriate level, and the configured threshold includes events at that level or higher while excluding less severe events.
- The default threshold and how an operator changes it are documented.
- Tests verify that events on both sides of a configured threshold are emitted or filtered as expected.

## Dependencies and open questions

- No dependencies.
- Choose the default threshold and configuration mechanism during implementation; neither is established by the current backlog.
