# Scope metadata names to their parent hierarchy

## Desired outcome

Interfaces, Objects, and Columns can reuse names in different parent hierarchies while remaining uniquely identifiable within their own hierarchy.

## Scope

Change the metadata table structure and dependent relationships, backend, API, and editor so that identity and uniqueness follow the Source, Interface, Object, and Column hierarchy. Migrate existing installations without losing metadata or relationships. Preserve existing behavior where it does not conflict with scoped names.

## Acceptance criteria

- [x] Interface names are unique within a Source: the same name can be created under different Sources, but a duplicate under the same Source is rejected.
- [x] Object names are unique within a Source and Interface: the same name can be created under different Interfaces, including same-named Interfaces in different Sources, but a duplicate in the same Source and Interface is rejected.
- [x] Column names are unique within a Source, Interface, and Object: the same name can be created under different Objects, including same-named Objects in different hierarchies, but a duplicate in the same hierarchy is rejected.
- [x] Database constraints enforce these rules and ensure Objects, Columns, and API parameters reference the intended parent even when names are repeated. Missing or referenced parents continue to be handled safely.
- [x] An upgrade of an existing database preserves all metadata and relationships, including API parameter references, without resetting the volume. Repeating startup or migration preserves the migrated data.
- [x] API and editor list, create, retrieve, edit, delete, and parent selection operations distinguish same-named records by their full context, act on the intended record, and surface conflicts. Parent changes preserve relationships and reject conflicts rather than affecting another hierarchy.
- [x] Metadata audit entries for new changes distinguish same-named records, and existing audit history remains available.
- [x] Focused backend, frontend, and real MariaDB integration tests cover allowed and rejected duplicates at each level, unambiguous CRUD and relationships, conflicts, and migration of an existing hierarchy. User and technical documentation describes the new identity and any API or upgrade changes.

## Decisions and implementation

- Source names remain source-system identities. Newly created Interfaces, Objects, and Columns use immutable server-generated UUID identifiers; migrated records retain their former name as their stable ID. Names are unique respectively by `(source, interface name)`, `(interface id, object name)`, and `(object id, column name)`.
- API references, lookups, editor operations, and new audit entries use stable IDs. POST responses provide generated IDs, and parent references use `interfaceId` or `objectId`. This changes the API record shapes; existing clients must move to the new properties.
- Liquibase change set 005 backfills the legacy globally unique names as IDs, preserves parent links and API parameters, retains historical audit rows, then installs the new keys and constraints. The migration is repeatable and does not reset database volumes.
- Reparenting changes the parent ID while keeping the record ID and dependent records stable. Database unique constraints reject destination conflicts. New audit details include the stable ID and current names/context; existing audit entries remain unchanged and available.
