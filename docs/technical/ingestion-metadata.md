# Data ingestion metadata model

This document describes the logical model for ingestion metadata and its
implemented database schema, Spring Boot object model, REST API, and frontend.
Liquibase owns the schema; Spring Data JDBC maps immutable Java records to the
six tables. References are stored as foreign-key strings, not nested objects.

## Entities

### Source system

| Property | Key | Description | Nullable | Datatype |
| --- | --- | --- | -- | -- |
| Source name | Primary key | Identifies the source system. | false | text |
| Source functional name | | Business-facing name of the source system. | true | text |
| Source type | | Kind of source system. | false | text |
| Source Owner | | The primary contact person for the source system | true | text |
| Source application | | Name of the application this source belongs to | false | text |
| Source Location | | Where to find this source. Content depends on Source type. E.g. an API would have the API base URL here | true | text |
| Is Active | | Whether this source should be processed or not | false | boolean |

### Source system interface

| Property | Key | Description | Nullable | Datatype |
| --- | --- | --- | -- | -- |
| Source name | Foreign key | References the source system's source name. | false | text |
| Interface id | Primary key | Stable identity for the interface. | false | text |
| Interface name | Scoped unique name | Identifies the interface within its source system. | false | text |
| Authentication id | Foreign key | References an optional authentication record's authentication id. | true | text |
| Interface functional name | | Business-facing name of the interface. | true | text |
| Interface type | | Kind of interface. | false | text |
| Interface location | | Location at which the interface is accessed. | false | text |
| Decimal separator | | Character used as decimal separator | false | single character |
| Thousands separator | | Character used as thousands separator | false | single character |
| Delimiter | | Character used as delimiter in flatfile use | true | single character |
| HasHeaders | | Whether a flatfile interface type includes a header row | true | boolean |
| Interface Location | | Where to find this interface within the source location. E.g. API endpoint | true | text |
| Is Active | | Whether this interface should be processed or not | false | boolean |

### Source object

| Property | Key | Description | Nullable | Datatype |
| --- | --- | --- | -- | -- |
| Object id | Primary key | Stable identity for the source object. | false | text |
| Object name | Scoped unique name | Identifies the object within its interface. | false | text |
| Interface id | Foreign key | References the interface that owns the object. | false | text |
| Object functional name | | Business-facing name of the object. | true | text |
| Object Description | | Description of the content of this object | true | text |
| IsConfidential | | Whether this object contains confidential information | false | boolean |
| IsBusinessCritical | | Whether this object contains business critical information | false | boolean |
| Is Active | | Whether this object should be processed or not | false | boolean |

### Source column

| Property | Key | Description | Nullable | Datatype |
| --- | --- | --- | -- | -- |
| Column id | Primary key | Stable identity for the column. | false | text |
| Column name | Scoped unique name | Identifies the column within its object. | false | text |
| Object id | Foreign key | References the object that owns the column. | false | text |
| Column alternate name | | What the column is called in the source system, if different from the Column Name | true | text |
| Column functional name | | Business-facing name of the column. | true | text |
| Column Description | | Description of the content of the column | true | text |
| Column ordinal position | | Position of this column in the target table | false | integer |
| Needs Encryption | | Whether the column needs to be encrypted on storage | false | boolean |
| Is partition field | | Whether the target storage needs to be partitioned on this column | false | boolean |
| Partition ordinal position | What position this column takes in the partitioning | false | integer |
| Is Key | | Whether this column contains a (part of the) key for this object | false | boolean |
| Key ordinal position | | What position this column takes in the key | false | integer |
| Is Nullable | | Whether NULL values are accepted in this column | false | boolean |
| Test Datatype | | Whether to test the datatype constraints on ingestion | false | boolean |
| Datatype | | Source datatype of the column. | false | text |
| Character maximum length | | On a text field: Maximum number of chaaracters | true | integer |
| Numeric scale | | How many total digits in a numeric value | true | integer |
| Numeric precision | | How many digits after the decimal separator | true | integer |
| Null value regex pattern | | If the value matches this regex pattern, it is replaced by NULL | true | text |
| Is Active | | Whether this column should be processed or not | false | boolean |

### Authentication

| Property | Key | Description | Nullable | Datatype |
| --- | --- | --- | -- | -- |
| Authentication id | Primary key | Identifies the authentication record. | false | text |
| Authentication type | | Kind of authentication. | false | text |
| Authentication location | | Location of the authentication material. | false | text |
| Authentication user | | Username to authenticate with | true | text |
| Authentication credential | | Credential to authenticate with | true | text |
| Is Active | | Whether this authentication method is active or not | false | boolean |

Location contains different information based on Authentication type. For example,
with Authentication type 'Bearer token' the location is the URL to call to retrieve
the token; with 'license file' it is the local folder and filename of the license file.

### API parameters

| Property | Key | Description | Nullable | Datatype |
| --- | --- | --- | -- | -- |
| Parameter id | Primary key | Identifies the API parameter. | false | text |
| Interface id | Foreign key | References the interface that uses this parameter. | false | text |
| Parameter name | | Name of the parameter | false | text |
| Parameter value | | Value of the parameter to be passed to the API | true | text |

## Relationships and identity

- A source system can have multiple interfaces; each interface references its
  source system.
- An interface can have multiple source objects; each object references its
  interface.
- An object can have multiple columns; each column references its object.
- An interface's authentication id references an authentication record.
  Multiple interfaces may reference the same record; authentication is optional
  for each interface in this implementation.
- Multiple API parameters can reference the same Interface.

Sources retain their source name as identity. Interfaces, objects, and columns
have immutable server-generated string IDs; their names are unique respectively
within a source, interface, and object. API parameters and objects reference an
interface ID, while columns reference an object ID. Reparenting changes the
parent reference without changing the record's ID or its children's references.

## Physical schema and validation

Tables are `source_system`, `source_system_interface`, `source_object`,
`source_column`, `authentication`, and `api_parameter`. Columns use snake_case
equivalents of the properties above. Primary keys are immutable `VARCHAR(255)`
strings; newly created interface, object, and column IDs are UUIDs, while
migrated records retain their old name as their stable ID. Composite unique
constraints enforce scoped names, and foreign keys use stable IDs to identify
the exact parent. Non-nullable text fields must be nonblank; optional
fields may be null, not empty. Descriptive fields and references have a
255-character maximum; interface and authentication locations allow 2048.
Types and datatypes are free text, not predefined enums. Locations are metadata;
the application does not access them. Authentication user and credential values,
and API parameter values, are stored unencrypted as supplied and returned by the
REST API. The API has no login or authorization layer, so only use trusted
local-development values; do not store production secrets.

Database constraints enforce scoped name uniqueness. Foreign keys are indexed
and reject missing parents and deletion of referenced records.
There is no cascading deletion: delete columns, then objects and API parameters,
then interfaces, then their source systems or authentication references. Updates may change
descriptions and references, but not primary keys. Edits use last-write-wins;
there is no record version or multi-user edit lock. Names for interfaces,
objects, and columns may change when the new scoped name is available; IDs
remain immutable.

Migrations live in `backend/src/main/resources/db/changelog/`. Spring Boot runs
the master changelog before serving traffic, using the configured database
account. Change set 005 backfills stable IDs from the previously globally unique
names, rewires object, column, and API-parameter relationships, and retains all
metadata and audit history. Existing audit record IDs remain unchanged; new
entries use the stable ID and include names and parent context in their details.
Existing database volumes are upgraded automatically;
no recreation or data reset is needed. Liquibase's history and lock tables prevent
reapplying changesets. Add new changesets rather than editing deployed ones.
The initial changeset includes a reverse-order rollback, which destroys the
metadata tables and must only be used intentionally with a backup.

## REST API

All resources are under `/api/metadata`:

| Resource | JSON properties |
| --- | --- |
| `source-systems` | `sourceName`, `sourceFunctionalName`, `sourceType`, `sourceOwner`, `sourceApplication`, `sourceLocation`, `isActive` |
| `authentications` | `authenticationId`, `authenticationType`, `authenticationLocation`, `authenticationUser`, `authenticationCredential`, `isActive` |
| `api-parameters` | `parameterId`, `interfaceId`, `parameterName`, `parameterValue` |
| `interfaces` | `interfaceId`, `interfaceName`, `sourceName`, `authenticationId`, `interfaceFunctionalName`, `interfaceType`, `interfaceLocation`, `decimalSeparator`, `thousandsSeparator`, `delimiter`, `hasHeaders`, `isActive` |
| `objects` | `objectId`, `objectName`, `interfaceId`, `objectFunctionalName`, `objectDescription`, `isConfidential`, `isBusinessCritical`, `isActive` |
| `columns` | `columnId`, `columnName`, `objectId`, `columnAlternateName`, `columnFunctionalName`, `columnDescription`, `columnOrdinalPosition`, `needsEncryption`, `isPartitionField`, `partitionOrdinalPosition`, `isKey`, `keyOrdinalPosition`, `isNullable`, `testDatatype`, `datatype`, `characterMaximumLength`, `numericScale`, `numericPrecision`, `nullValueRegexPattern`, `isActive` |

| Method and URL | Behavior |
| --- | --- |
| `GET /api/metadata/{resource}` | Return a JSON array sorted by primary key (empty array if none). |
| `GET /api/metadata/{resource}?id={encoded-id}` | Return one record or HTTP 404. |
| `POST /api/metadata/{resource}` | Create from a complete JSON object; HTTP 201 with body and `Location` header. |
| `PUT /api/metadata/{resource}?id={encoded-id}` | Replace an existing record with the complete JSON object; HTTP 200 with body. |
| `DELETE /api/metadata/{resource}?id={encoded-id}` | Delete an unreferenced record; HTTP 204 with no body. |

The query parameter avoids path-separator restrictions for identifiers. URL-encode
IDs, including spaces, `+`, `&`, `?`, and `#`. Interface POST requests omit
`interfaceId`, object POST requests omit `objectId`, and column POST requests
omit `columnId`; the server generates and returns these IDs. Objects and API
parameters identify their parent by `interfaceId`; columns identify their
parent by `objectId`. Omitted optional authentication is stored and returned as
null. Unknown JSON properties,
missing/blank required fields, excessive lengths, and attempted ID changes
return HTTP 400. Duplicate scoped names, missing foreign keys, and referenced deletions
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

Java integration tests run the Liquibase schema on H2 in MariaDB mode and
exercise CRUD, validation boundaries, scoped names, relationships, nullable/shared
authentication, API parameters, reparenting, existing-schema migration, and Problem Detail failures. Angular tests exercise
the editor's views, HTTP operations, selectors, validation, confirmation, and
error/retry states. Docker tests exercise all six resources through Nginx on
real MariaDB, then restart the backend to verify migration idempotence and data
persistence. Run all suites with `make test`; see [setup](setup.md).

See the [frontend user guide](../user/ingestion-metadata.md) for creating and
editing metadata.
