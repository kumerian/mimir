# Table: `object`

Canonical definition of the `object` metadata table.
This file is the authoritative reference for the table's schema.
All creation, reading, and update operations in code must reflect this definition.

## Purpose

Tracks registered data objects (e.g. tables, files, APIs) that belong to a source.
Each row describes one logical object within a source system.

## Columns

| Column      | Type    | Nullable | Default | Primary Key | Description                                                              |
|-------------|---------|----------|---------|-------------|--------------------------------------------------------------------------|
| name        | string  | NO       | —       | YES         | Unique identifier for the object (e.g. `crm_salesforce.contacts`).       |
| source_name | string  | NO       | —       | NO          | Foreign key reference to the `source` table.                             |
| type        | string  | YES      | —       | NO          | Category of the object (e.g. `table`, `view`, `file`, `endpoint`).       |
| description | string  | YES      | —       | NO          | Human-readable description of the object and its contents.               |
| is_active   | boolean | NO       | TRUE    | NO          | Whether the object is currently enabled for ingestion.                   |
| is_current  | boolean | NO       | TRUE    | NO          | Whether this row is the current (non-superseded) version.                |

## Constraints

- `name` is the primary key; values must be unique and non-null.
- `source_name` is a foreign key referencing `source(name)`; must be non-null.
- `is_active` and `is_current` must never be NULL; default to `TRUE`.

## Notes

- `is_current` supports a soft-versioning pattern: set to `FALSE` to deprecate a row without deleting it.
