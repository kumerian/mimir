# Audit metadata changes made through the web interface

## Desired outcome

Changes made to ingestion metadata through the web interface leave a durable audit trail that can be inspected through the interface.

## Scope

Cover successful metadata creation, editing, and deletion through the existing metadata editor. Keep this audit trail distinct from diagnostic application logs. The current application has no login or authorization layer, so this item does not assume authenticated user identities.

## Acceptance criteria

- Each successful create, update, or delete through the web interface produces an audit entry; rejected or failed writes are not presented as successful changes.
- Entries identify the operation, metadata entity and record, and time of the change, and provide enough detail to determine what changed.
- Authentication credentials and API parameter values are excluded or redacted from audit entries.
- Audit entries are stored durably and can be viewed through the web interface.
- Tests cover create, update, and delete entries, failed writes, and sensitive-value redaction.

## Dependencies and open questions

- No dependency on general database logging is assumed; audit history is a distinct product record, even if it shares database infrastructure.
- Decide the audit detail and retention policy during implementation. Actor attribution is unresolved because the current application has no authentication; do not imply that an entry identifies a person unless that capability is separately established.
