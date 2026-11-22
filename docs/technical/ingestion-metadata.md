# Data ingestion metadata model

This document describes the logical model for ingestion metadata and its
implemented database schema, Spring Boot object model, REST API, and frontend.
Liquibase owns the schema; Spring Data JDBC maps immutable Java records to the
five tables. References are stored as foreign-key strings, not nested objects.

## Entities

### Source system

| Property | Key | Description |
| --- | --- | --- |
| Source name | Primary key | Identifies the source system. |
| Source functional name | | Business-facing name of the source system. |
| Source type | | Kind of source system. |

### Source system interface

| Property | Key | Description |
| --- | --- | --- |
| Source name | Foreign key | References the source system's source name. |
| Interface name | Primary key | Identifies the interface. |
| Authentication id | Foreign key | References an authentication record's authentication id. |
| Interface functional name | | Business-facing name of the interface. |
| Interface type | | Kind of interface. |
| Interface location | | Location at which the interface is accessed. |

### Source object

| Property | Key | Description |
| --- | --- | --- |
| Interface name | Foreign key | References the source system interface's interface name. |
| Object name | Primary key | Identifies the source object, such as a table. |
| Object functional name | | Business-facing name of the object. |

### Source column

| Property | Key | Description |
| --- | --- | --- |
| Object name | Foreign key | References the source object's object name. |
| Column name | Primary key | Identifies the column. |
| Column functional name | | Business-facing name of the column. |
| Datatype | | Source datatype of the column. |

### Authentication

| Property | Key | Description |
| --- | --- | --- |
| Authentication id | Primary key | Identifies the authentication record. |
| Authentication type | | Kind of authentication. |
| Authentication location | | Location of the authentication material. |

Authentication metadata identifies where authentication material is located;
it does not contain the material itself.

## Relationships and identity

- A source system can have multiple interfaces; each interface references its
  source system.
- An interface can have multiple source objects; each object references its
  interface.
- An object can have multiple columns; each column references its object.
- An interface's authentication id references an authentication record.
  Multiple interfaces may reference the same record; authentication is optional
  for each interface in this implementation.

The specified primary keys are individual names, not composite keys. Interface,
object, and column names must therefore each be unique across all records of
their respective entity, not just within their parent.

## Physical schema and validation

Tables are `source_system`, `source_system_interface`, `source_object`,
`source_column`, and `authentication`. Columns use snake_case equivalents of
the properties above. Every primary key is an immutable `VARCHAR(255)` string.
All fields are required and nonblank except an interface's `authentication_id`,
which may be null (not an empty string). Descriptive fields and references have
a 255-character maximum; interface and authentication locations allow 2048.
Types and datatypes are free text, not predefined enums. Locations are metadata;
the application does not access them or store credential material.

Database primary keys enforce global uniqueness within each table. Foreign keys
are indexed and reject missing parents and deletion of referenced records.
There is no cascading deletion: delete columns, then objects, then interfaces,
then their source systems or authentication references. Updates may change
descriptions and references, but not primary keys. Edits use last-write-wins;
there is no record version or multi-user edit lock.

Migrations live in `backend/src/main/resources/db/changelog/`. Spring Boot runs
the master changelog before serving traffic, using the configured database
account. Existing database volumes are upgraded automatically; no recreation
or data reset is needed. Liquibase's history and lock tables prevent reapplying
the initial changeset. Add new changesets rather than editing deployed ones.
The initial changeset includes a reverse-order rollback, which destroys the
metadata tables and must only be used intentionally with a backup.

## REST API

All resources are under `/api/metadata`:

| Resource | JSON properties |
| --- | --- |
| `source-systems` | `sourceName`, `sourceFunctionalName`, `sourceType` |
| `authentications` | `authenticationId`, `authenticationType`, `authenticationLocation` |
| `interfaces` | `interfaceName`, `sourceName`, `authenticationId`, `interfaceFunctionalName`, `interfaceType`, `interfaceLocation` |
| `objects` | `objectName`, `interfaceName`, `objectFunctionalName` |
| `columns` | `columnName`, `objectName`, `columnFunctionalName`, `datatype` |

| Method and URL | Behavior |
| --- | --- |
| `GET /api/metadata/{resource}` | Return a JSON array sorted by primary key (empty array if none). |
| `GET /api/metadata/{resource}?id={encoded-key}` | Return one record or HTTP 404. |
| `POST /api/metadata/{resource}` | Create from a complete JSON object; HTTP 201 with body and `Location` header. |
| `PUT /api/metadata/{resource}?id={encoded-key}` | Replace an existing record with the complete JSON object; HTTP 200 with body. |
| `DELETE /api/metadata/{resource}?id={encoded-key}` | Delete an unreferenced record; HTTP 204 with no body. |

The query parameter avoids path-separator restrictions for names containing `/`.
URL-encode key values, including spaces, `+`, `&`, `?`, and `#`. Omitted optional
authentication is stored and returned as null. Unknown JSON properties,
missing/blank required fields, excessive lengths, and attempted key changes
return HTTP 400. Duplicate keys, missing foreign keys, and referenced deletions
return HTTP 409. Missing records return HTTP 404. Errors use Spring Problem
Detail JSON with `status` and `detail`; persistence failures return HTTP 500,
are logged, and do not claim success or disclose SQL in the response.

Example creation through the frontend proxy:

```sh
curl -i -X POST http://localhost:8080/api/metadata/source-systems \
  -H 'Content-Type: application/json' \
  -d '{"sourceName":"crm","sourceFunctionalName":"Customer management","sourceType":"database"}'
curl 'http://localhost:8080/api/metadata/source-systems?id=crm'
```

The API has no login or authorization layer. Keep the default loopback binding;
add deployment access controls before use on shared or public infrastructure.
This implementation manages metadata only, not ingestion execution.

## Tests and user instructions

Java integration tests run the real Liquibase schema on H2 in MariaDB mode and
exercise CRUD, validation boundaries, global keys, relationships, nullable/shared
authentication, reparenting, and Problem Detail failures. Angular tests exercise
the editor's views, HTTP operations, selectors, validation, confirmation, and
error/retry states. Docker tests exercise all five resources through Nginx on
real MariaDB, then restart the backend to verify migration idempotence and data
persistence. Run all suites with `make test`; see [setup](setup.md).

See the [frontend user guide](../user/ingestion-metadata.md) for creating and
editing metadata.
