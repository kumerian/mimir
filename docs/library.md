# Mimir Library

Mimir is a Python library for metadata-driven ingestion workflows.

## Public API

- `ensure_reference_database(db_path=None)`: creates and initializes DuckDB schema.
- `get_reference_db_path(db_path=None)`: resolves the reference DB path.
- `register_source(source_name, source_type, location, db_path=None)`: upserts metadata source information.
- `list_sources(db_path=None)`: returns registered metadata sources sorted by name.

## Reference Database

The library uses DuckDB as a lightweight local metadata store.

Default path:

- `data/reference.duckdb`

Schema source:

- `sql/reference_schema.sql`
