# Table: `source`

Canonical definition of the `source` metadata table.
This file is the authoritative reference for the table's schema.
All creation, reading, and update operations in code must reflect this definition.

## Purpose

Tracks registered data sources that Mimir ingestion workflows operate against.
Each row describes one logical source system or feed.

## Columns

| Column      | Type    | Nullable | Default | Primary Key | Description                                                     |
|-------------|---------|----------|---------|-------------|------------------------------------------------------------------|
| name        | string  | NO       | —       | YES         | Unique identifier for the source (e.g. `crm_salesforce`).       |
| type        | string  | YES      | —       | NO          | Category of the source (e.g. `database`, `api`, `file`).        |
| system      | string  | YES      | —       | NO          | Underlying platform or technology (e.g. `mssql`, `rest`).       |
| owner       | string  | YES      | —       | NO          | Team or individual responsible for this source.                  |
| timezone    | string  | YES      | —       | NO          | IANA timezone string for the source (e.g. `Europe/Amsterdam`).  |
| is_active   | boolean | NO       | TRUE    | NO          | Whether the source is currently enabled for ingestion.           |
| is_current  | boolean | NO       | TRUE    | NO          | Whether this row is the current (non-superseded) version.        |

## Constraints

- `name` is the primary key; values must be unique and non-null.
- `is_active` and `is_current` must never be NULL; default to `TRUE`.

## Notes

- `timezone` follows the IANA Time Zone Database format (e.g. `UTC`, `America/New_York`).
- `is_current` supports a soft-versioning pattern: set to `FALSE` to deprecate a row without deleting it.
