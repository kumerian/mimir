"""Object table definition, record model, and repository.

The canonical column specification lives in:
    documentation/tables/object.md

``OBJECT_TABLE`` is built at import time by parsing that markdown file.
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
# Canonical table definition — loaded from metadata/metadata/object.md
# ---------------------------------------------------------------------------

# Locate the markdown file relative to the repository root.
# This file lives at: src/mimir/metadata/object_repository.py
# Repo root is 3 levels up:  metadata/ -> mimir/ -> src/ -> repo root
_OBJECT_MD: Path = Path(__file__).parents[3] / "metadata" / "metadata" / "object.md"

OBJECT_TABLE: TableDefinition = load_table_definition(_OBJECT_MD)

# Columns that may be updated after insertion (primary key and FK excluded).
_UPDATABLE_COLUMNS: frozenset[str] = frozenset(
    {"type", "description", "is_active", "is_current"}
)


# ---------------------------------------------------------------------------
# Record model
# ---------------------------------------------------------------------------


@dataclass
class ObjectRecord:
    """Represents a single row in the ``object`` table."""

    name: str
    source_name: str
    type: str | None = None
    description: str | None = None
    is_active: bool = field(default=True)
    is_current: bool = field(default=True)

    def to_dict(self) -> dict[str, Any]:
        """Return a plain dictionary representation suitable for ``write_table``."""
        return {
            "name": self.name,
            "source_name": self.source_name,
            "type": self.type,
            "description": self.description,
            "is_active": self.is_active,
            "is_current": self.is_current,
        }

    @classmethod
    def from_dict(cls, row: dict[str, Any]) -> "ObjectRecord":
        """Construct an ``ObjectRecord`` from a row dictionary returned by ``read_table``."""
        return cls(
            name=row["name"],
            source_name=row["source_name"],
            type=row.get("type"),
            description=row.get("description"),
            is_active=bool(row["is_active"]),
            is_current=bool(row["is_current"]),
        )


# ---------------------------------------------------------------------------
# Repository
# ---------------------------------------------------------------------------


class ObjectRepository:
    """Create, read, and update rows in the ``object`` metadata table.

    The table schema is governed by ``metadata/metadata/object.md`` and
    mirrored in :data:`OBJECT_TABLE`.
    """

    def __init__(self, client: DatabaseClient) -> None:
        self._client = client

    # ------------------------------------------------------------------
    # DDL
    # ------------------------------------------------------------------

    def create_table(self, if_not_exists: bool = True) -> str:
        """Create the ``object`` table and return the executed SQL."""
        creator = MetadataTableCreator(self._client)
        return creator.create_table(OBJECT_TABLE, if_not_exists=if_not_exists)

    # ------------------------------------------------------------------
    # Write
    # ------------------------------------------------------------------

    def insert(self, record: ObjectRecord) -> None:
        """Insert a new object record.

        Raises :exc:`ValueError` if a record with the same name already exists.
        """
        if self.get_by_name(record.name) is not None:
            raise ValueError(f"An object record with name '{record.name}' already exists.")
        self._client.write_table("object", [record.to_dict()])

    def update(self, name: str, **fields: Any) -> bool:
        """Update one or more fields for the object record identified by *name*.

        Only columns listed in ``_UPDATABLE_COLUMNS`` may be updated.
        Returns ``True`` when a matching record was found and updated,
        ``False`` otherwise.

        Raises :exc:`ValueError` for unknown or non-updatable column names.
        """
        unknown = set(fields) - _UPDATABLE_COLUMNS
        if unknown:
            raise ValueError(
                f"Unknown or non-updatable field(s): {sorted(unknown)}. "
                f"Allowed: {sorted(_UPDATABLE_COLUMNS)}"
            )
        if not fields:
            return False

        if self.get_by_name(name) is None:
            return False

        set_clause = ", ".join(f"{col} = ?" for col in fields)
        params: list[Any] = list(fields.values()) + [name]
        self._client.execute(
            f"UPDATE object SET {set_clause} WHERE name = ?",  # noqa: S608
            params=params,
        )
        return True

    # ------------------------------------------------------------------
    # Read
    # ------------------------------------------------------------------

    def get_all(self) -> list[ObjectRecord]:
        """Return all rows from the ``object`` table."""
        rows = self._client.read_table("object")
        return [ObjectRecord.from_dict(row) for row in rows]

    def get_by_name(self, name: str) -> ObjectRecord | None:
        """Return the object record with the given *name*, or ``None``."""
        rows = self._client.read_table("object")
        for row in rows:
            if row["name"] == name:
                return ObjectRecord.from_dict(row)
        return None

    def get_by_source(self, source_name: str) -> list[ObjectRecord]:
        """Return all object records belonging to *source_name*."""
        return [record for record in self.get_all() if record.source_name == source_name]

    def get_active(self) -> list[ObjectRecord]:
        """Return all object records where ``is_active`` is ``True``."""
        return [record for record in self.get_all() if record.is_active]

    def get_current(self) -> list[ObjectRecord]:
        """Return all object records where ``is_current`` is ``True``."""
        return [record for record in self.get_all() if record.is_current]
