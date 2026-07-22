"""Core utilities for metadata-driven ingestion workflows."""

from __future__ import annotations

from pathlib import Path
from typing import Iterable

import duckdb


def _project_root() -> Path:
    return Path(__file__).resolve().parents[2]


def get_reference_db_path(db_path: str | Path | None = None) -> Path:
    """Return the absolute path to the reference DuckDB database."""
    if db_path is None:
        return _project_root() / "data" / "reference.duckdb"
    return Path(db_path).expanduser().resolve()


def _schema_sql_path() -> Path:
    return _project_root() / "sql" / "reference_schema.sql"


def ensure_reference_database(db_path: str | Path | None = None) -> Path:
    """Create and initialize the reference database when it does not exist."""
    resolved_db_path = get_reference_db_path(db_path)
    resolved_db_path.parent.mkdir(parents=True, exist_ok=True)

    with duckdb.connect(str(resolved_db_path)) as connection:
        schema_sql = _schema_sql_path().read_text(encoding="utf-8")
        connection.execute(schema_sql)

    return resolved_db_path


def register_source(
    source_name: str,
    source_type: str,
    location: str,
    db_path: str | Path | None = None,
) -> None:
    """Insert or update a source definition in the reference metadata DB."""
    resolved_db_path = ensure_reference_database(db_path)

    upsert_sql = """
        INSERT INTO metadata_sources (source_name, source_type, location)
        VALUES (?, ?, ?)
        ON CONFLICT(source_name)
        DO UPDATE SET
            source_type = excluded.source_type,
            location = excluded.location,
            updated_at = now()
    """

    with duckdb.connect(str(resolved_db_path)) as connection:
        connection.execute(upsert_sql, [source_name, source_type, location])


def list_sources(db_path: str | Path | None = None) -> Iterable[dict[str, str]]:
    """Return all registered metadata sources ordered by source name."""
    resolved_db_path = ensure_reference_database(db_path)
    query = """
        SELECT source_name, source_type, location
        FROM metadata_sources
        ORDER BY source_name
    """

    with duckdb.connect(str(resolved_db_path), read_only=True) as connection:
        rows = connection.execute(query).fetchall()

    return [
        {
            "source_name": row[0],
            "source_type": row[1],
            "location": row[2],
        }
        for row in rows
    ]
