# Audit metadata changes made through the web interface

## Desired outcome

Changes made to ingestion metadata through the web interface leave a durable audit trail that can be inspected through the interface.

## Scope

Cover successful metadata creation, editing, and deletion through the existing metadata editor. Keep this audit trail distinct from diagnostic application logs. The current application has no login or authorization layer, so this item does not assume authenticated user identities.

## Acceptance criteria

- [x] Each successful create, update, or delete through the web interface produces an audit entry; rejected or failed writes are not presented as successful changes.
- [x] Entries identify the operation, metadata entity and record, and time of the change, and provide enough detail to determine what changed.
- [x] Authentication credentials and API parameter values are excluded or redacted from audit entries.
- [x] Audit entries are stored durably and can be viewed through the web interface.
- [x] Tests cover create, update, and delete entries, failed writes, and sensitive-value redaction.

## Dependencies and implementation decisions

- No dependency on general database logging is assumed; audit history is a distinct product record, even if it shares database infrastructure.
- Audit details contain field-level before/after values, except authentication user/credential and API parameter values, which are redacted; entries are retained indefinitely.
- No actor is recorded because the application has no authentication.

## Implementation evidence

[MetadataAuditService](../../../backend/src/main/java/io/mimir/metadata/MetadataAuditService.java) records successful changes in the metadata transaction and redacts authentication user/credential and API parameter values. [MetadataAuditController](../../../backend/src/main/java/io/mimir/metadata/MetadataAuditController.java) and the [AuditLog UI](../../../frontend/src/app/audit/audit-log.ts) provide newest-first history. [MetadataApiTest](../../../backend/src/test/java/io/mimir/metadata/MetadataApiTest.java) covers create/update/delete, rejected duplicate writes, and redaction; [AuditLog tests](../../../frontend/src/app/audit/audit-log.spec.ts) cover display and load failures. [DockerStackTest](../../../tests/docker/test_stack.py) verifies audit history survives a backend restart. `make test` passed.
