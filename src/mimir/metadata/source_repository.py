"""Source table definition, record model, and repository.

The canonical column specification lives in:
    documentation/tables/source.md

``SOURCE_TABLE`` is built at import time by parsing that markdown file.
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
# Canonical table definition — loaded from metadata/metadata/source.md
# ---------------------------------------------------------------------------

# Locate the markdown file relative to the repository root.
# This file lives at: src/mimir/metadata/source_repository.py
# Repo root is 3 levels up:  metadata/ -> mimir/ -> src/ -> repo root
_SOURCE_MD: Path = Path(__file__).parents[3] / "metadata" / "metadata" / "source.md"

SOURCE_TABLE: TableDefinition = load_table_definition(_SOURCE_MD)

# Columns that may be updated after insertion (primary key excluded).
_UPDATABLE_COLUMNS: frozenset[str] = frozenset(
    {"type", "system", "owner", "timezone", "is_active", "is_current"}
)


# ---------------------------------------------------------------------------
# Record model
# ---------------------------------------------------------------------------


@dataclass
class SourceRecord:
    """Represents a single row in the ``source`` table."""

    name: str
    type: str | None = None
    system: str | None = None
    owner: str | None = None
    timezone: str | None = None
    is_active: bool = field(default=True)
    is_current: bool = field(default=True)

    def to_dict(self) -> dict[str, Any]:
        """Return a plain dictionary representation suitable for ``write_table``."""
        return {
            "name": self.name,
            "type": self.type,
            "system": self.system,
            "owner": self.owner,
            "timezone": self.timezone,
            "is_active": self.is_active,
            "is_current": self.is_current,
        }

    @classmethod
    def from_dict(cls, row: dict[str, Any]) -> "SourceRecord":
        """Construct a ``SourceRecord`` from a row dictionary returned by ``read_table``."""
        return cls(
            name=row["name"],
            type=row.get("type"),
            system=row.get("system"),
            owner=row.get("owner"),
            timezone=row.get("timezone"),
            is_active=bool(row["is_active"]),
            is_current=bool(row["is_current"]),
        )


# ---------------------------------------------------------------------------
# Repository
# ---------------------------------------------------------------------------


class SourceRepository:
    """Create, read, and update rows in the ``source`` metadata table.

    The table schema is governed by ``documentation/tables/source.md`` and
    mirrored in :data:`SOURCE_TABLE`.
    """

    def __init__(self, client: DatabaseClient) -> None:
        self._client = client

    # ------------------------------------------------------------------
    # DDL
    # ------------------------------------------------------------------

    def create_table(self, if_not_exists: bool = True) -> str:
        """Create the ``source`` table and return the executed SQL."""
        creator = MetadataTableCreator(self._client)
        return creator.create_table(SOURCE_TABLE, if_not_exists=if_not_exists)

    # ------------------------------------------------------------------
    # Write
    # ------------------------------------------------------------------

    def insert(self, record: SourceRecord) -> None:
        """Insert a new source record.

        Raises :exc:`ValueError` if a record with the same name already exists.
        """
        if self.get_by_name(record.name) is not None:
            raise ValueError(f"A source record with name '{record.name}' already exists.")
        self._client.write_table("source", [record.to_dict()])

    def update(self, name: str, **fields: Any) -> bool:
        """Update one or more fields for the source record identified by *name*.

        Only columns listed in the table definition (excluding the primary key)
        may be updated.  Returns ``True`` when a matching record was found and
        updated, ``False`` otherwise.

        Raises :exc:`ValueError` for unknown column names.
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
            f"UPDATE source SET {set_clause} WHERE name = ?",  # noqa: S608
            params=params,
        )
        return True

    # ------------------------------------------------------------------
    # Read
    # ------------------------------------------------------------------

    def get_all(self) -> list[SourceRecord]:
        """Return all rows from the ``source`` table."""
        rows = self._client.read_table("source")
        return [SourceRecord.from_dict(row) for row in rows]

    def get_by_name(self, name: str) -> SourceRecord | None:
        """Return the source record with the given *name*, or ``None``."""
        rows = self._client.read_table("source")
        for row in rows:
            if row["name"] == name:
                return SourceRecord.from_dict(row)
        return None

    def get_active(self) -> list[SourceRecord]:
        """Return all source records where ``is_active`` is ``True``."""
        return [record for record in self.get_all() if record.is_active]

    def get_current(self) -> list[SourceRecord]:
        """Return all source records where ``is_current`` is ``True``."""
        return [record for record in self.get_all() if record.is_current]
