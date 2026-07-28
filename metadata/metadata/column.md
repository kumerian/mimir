# Table: `columns`

Canonical definition of the `columns` metadata table.
This file is the authoritative reference for the table's schema.
All creation, reading, and update operations in code must reflect this definition.

## Purpose

Tracks column-level metadata for objects (e.g. columns in tables).
Each row describes one column within an object.

## Columns

| Column           | Type    | Nullable | Default | Primary Key | Description                                                             |
|------------------|---------|----------|---------|-------------|-------------------------------------------------------------------------|
| name             | string  | NO       | —       | YES         | Column name within the object (e.g. `customer_id`).                     |
| object_name      | string  | NO       | —       | YES         | Foreign key reference to the `object` table.                            |
| functional_name  | string  | YES      | —       | NO          | Human-friendly name or alias for the column.                            |
| description      | string  | YES      | —       | NO          | Human-readable description of the column's purpose and content.         |
| ordinal_position | integer | YES      | —       | NO          | Position of the column within the object (1-based ordering).            |
| is_metadata      | boolean | NO       | FALSE   | NO          | Whether this column contains metadata (rather than business data).      |
| data_type        | string  | YES      | —       | NO          | Data type of the column (e.g. `int`, `varchar`, `timestamp`).          |
| is_key           | boolean | NO       | FALSE   | NO          | Whether this column is part of a key (primary or unique).               |
| is_nullable      | boolean | NO       | TRUE    | NO          | Whether the column allows NULL values.                                  |
| is_current       | boolean | NO       | TRUE    | NO          | Whether this row is the current (non-superseded) version.               |
| is_active        | boolean | NO       | TRUE    | NO          | Whether the column is currently enabled for ingestion.                  |

## Constraints

- `object_name` is the primary key and a foreign key referencing `object(name)`; must be non-null.
- `name` and `object_name` together form a unique composite key; each column name must be unique within an object.
- `is_metadata`, `is_key`, `is_nullable`, `is_active`, and `is_current` must never be NULL; they have appropriate defaults.

## Notes

- The composite key (name, object_name) is enforced by the fact that `object_name` is the primary key and columns are only added in the context of an object, ensuring uniqueness.
- `ordinal_position` reflects the logical order of columns as they appear in the source object.
- `is_current` supports a soft-versioning pattern: set to `FALSE` to deprecate a row without deleting it.
- `is_metadata` distinguishes metadata columns from business data columns.
