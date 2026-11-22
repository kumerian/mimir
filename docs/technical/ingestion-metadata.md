# Data ingestion metadata model

This document describes the logical model for ingestion metadata. It is a
design, not an implemented database schema; physical data types, field lengths,
nullability, and deletion behavior remain to be defined.

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
  Multiple interfaces may reference the same record; whether authentication is
  required for every interface remains to be defined.

The specified primary keys are individual names, not composite keys. Interface,
object, and column names must therefore each be unique across all records of
their respective entity, not just within their parent.
