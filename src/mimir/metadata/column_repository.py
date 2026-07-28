"""Column table definition, record model, and repository.

The canonical column specification lives in:
    documentation/tables/column.md

``COLUMN_TABLE`` is built at import time by parsing that markdown file.
Any schema change must be made in the markdown; the code will reflect it
automatically.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

from mimir.utility import DatabaseClient
from mimir.utility.markdown_parser import load_table_definition

from .schema import TableDefinition
from .table_creator import MetadataTableCreator

# ---------------------------------------------------------------------------
# Canonical table definition — loaded from metadata/metadata/column.md
# ---------------------------------------------------------------------------

# Locate the markdown file relative to the repository root.
# This file lives at: src/mimir/metadata/column_repository.py
# Repo root is 3 levels up:  metadata/ -> mimir/ -> src/ -> repo root
_COLUMN_MD: Path = Path(__file__).parents[3] / "metadata" / "metadata" / "column.md"

COLUMN_TABLE: TableDefinition = load_table_definition(_COLUMN_MD)
# Columns that may be updated after insertion (primary key and FK excluded).
_UPDATABLE_COLUMNS: frozenset[str] = frozenset(
    {
        "functional_name",
        "description",
        "ordinal_position",
        "is_metadata",
        "data_type",
        "is_key",
        "is_nullable",
        "is_current",
        "is_active",
    }
)


# ---------------------------------------------------------------------------
# Record model
# ---------------------------------------------------------------------------


@dataclass
class ColumnRecord:
    """Represents a single row in the ``column`` table."""

    name: str
    object_name: str
    functional_name: str | None = None
    description: str | None = None
    ordinal_position: int | None = None
    is_metadata: bool = field(default=False)
    data_type: str | None = None
    is_key: bool = field(default=False)
    is_nullable: bool = field(default=True)
    is_current: bool = field(default=True)
    is_active: bool = field(default=True)

    def to_dict(self) -> dict[str, Any]:
        """Return a plain dictionary representation suitable for ``write_table``."""
        return {
            "name": self.name,
            "object_name": self.object_name,
            "functional_name": self.functional_name,
            "description": self.description,
            "ordinal_position": self.ordinal_position,
            "is_metadata": self.is_metadata,
            "data_type": self.data_type,
            "is_key": self.is_key,
            "is_nullable": self.is_nullable,
            "is_current": self.is_current,
            "is_active": self.is_active,
        }

    @classmethod
    def from_dict(cls, row: dict[str, Any]) -> "ColumnRecord":
        """Construct a ``ColumnRecord`` from a row dictionary returned by ``read_table``."""
        return cls(
            name=row["name"],
            object_name=row["object_name"],
            functional_name=row.get("functional_name"),
            description=row.get("description"),
            ordinal_position=row.get("ordinal_position"),
            is_metadata=bool(row.get("is_metadata", False)),
            data_type=row.get("data_type"),
            is_key=bool(row.get("is_key", False)),
            is_nullable=bool(row.get("is_nullable", True)),
            is_current=bool(row.get("is_current", True)),
            is_active=bool(row.get("is_active", True)),
        )


# ---------------------------------------------------------------------------
# Repository
# ---------------------------------------------------------------------------


class ColumnRepository:
    """Create, read, and update rows in the ``columns`` metadata table.

    The table schema is governed by ``metadata/metadata/column.md`` and
    mirrored in :data:`COLUMN_TABLE`.
    """

    def __init__(self, client: DatabaseClient) -> None:
        self._client = client

    # ------------------------------------------------------------------
    # DDL
    # ------------------------------------------------------------------

    def create_table(self, if_not_exists: bool = True) -> str:
        """Create the ``columns`` table and return the executed SQL."""
        creator = MetadataTableCreator(self._client)
        return creator.create_table(COLUMN_TABLE, if_not_exists=if_not_exists)

    # ------------------------------------------------------------------
    # DML
    # ------------------------------------------------------------------

    def insert_record(self, record: ColumnRecord) -> None:
        """Insert a single column record into the database."""
        self._client.write_table("columns", [record.to_dict()])

    def insert_records(self, records: list[ColumnRecord]) -> None:
        """Insert multiple column records into the database."""
        if records:
            data = [r.to_dict() for r in records]
            self._client.write_table("columns", data)

    def get_all_columns(self) -> list[ColumnRecord]:
        """Retrieve all column records from the database."""
        rows = self._client.read_table("columns")
        return [ColumnRecord.from_dict(row) for row in rows]

    def get_columns_for_object(self, object_name: str) -> list[ColumnRecord]:
        """Retrieve all column records for a specific object."""
        all_columns = self.get_all_columns()
        return [col for col in all_columns if col.object_name == object_name]

    def update_record(self, name: str, object_name: str, **fields: Any) -> bool:
        """Update a column record.

        Returns True if updated, False if not found.
        Raises ValueError for unknown or non-updatable columns.
        """
        unknown = set(fields) - _UPDATABLE_COLUMNS
        if unknown:
            raise ValueError(
                f"Unknown or non-updatable field(s): {sorted(unknown)}. "
                f"Allowed: {sorted(_UPDATABLE_COLUMNS)}"
            )
        if not fields:
            return False

        set_clause = ", ".join(f"{col} = ?" for col in fields)
        params: list[Any] = list(fields.values()) + [name, object_name]
        self._client.execute(
            f"UPDATE columns SET {set_clause} WHERE name = ? AND object_name = ?",  # noqa: S608
            params=params,
        )
        return True

    def delete_record(self, name: str, object_name: str) -> None:
        """Mark a column record as inactive and not current."""
        self.update_record(name, object_name, is_active=False, is_current=False)
