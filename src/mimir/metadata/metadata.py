"""Hierarchical metadata tree structure for metadata-driven ingestion.

This module provides a tree-based abstraction over the metadata tables:
- Metadata: top-level container representing the metadata database
- Source: represents a data source, created from a Metadata object
- Object: represents an object within a source, created from a Source object

The tree structure enables intuitive navigation and management of metadata
while abstracting away the underlying database operations.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import TYPE_CHECKING, Any

from mimir.utility import DatabaseClient

if TYPE_CHECKING:
    pass


# ---------------------------------------------------------------------------
# Object (leaf node)
# ---------------------------------------------------------------------------


@dataclass
class Object:
    """Represents a metadata object (table, view, etc.) within a source.

    An Object is always associated with a Source and cannot exist independently.
    It contains metadata about the object such as its name, type, and description.
    """

    name: str
    source: Source
    type: str | None = None
    description: str | None = None
    is_active: bool = field(default=True)
    is_current: bool = field(default=True)

    def to_dict(self) -> dict[str, Any]:
        """Return a dictionary representation suitable for database storage."""
        return {
            "name": self.name,
            "source_name": self.source.name,
            "type": self.type,
            "description": self.description,
            "is_active": self.is_active,
            "is_current": self.is_current,
        }

    def update(self, **fields: Any) -> bool:
        """Update metadata for this object.

        Returns True if updated, False if not found.
        Raises ValueError for unknown or non-updatable columns.
        """
        return self.source.metadata._update_object(self.name, **fields)

    def delete(self) -> None:
        """Mark this object as inactive and not current."""
        self.is_active = False
        self.is_current = False
        self.update(is_active=False, is_current=False)


# ---------------------------------------------------------------------------
# Source (intermediate node)
# ---------------------------------------------------------------------------


@dataclass
class Source:
    """Represents a data source (database, API, file, etc.).

    A Source is always associated with a Metadata and contains zero or more Objects.
    """

    name: str
    metadata: Metadata
    type: str | None = None
    system: str | None = None
    owner: str | None = None
    timezone: str | None = None
    is_active: bool = field(default=True)
    is_current: bool = field(default=True)
    objects: dict[str, Object] = field(default_factory=dict, init=False)

    def to_dict(self) -> dict[str, Any]:
        """Return a dictionary representation suitable for database storage."""
        return {
            "name": self.name,
            "type": self.type,
            "system": self.system,
            "owner": self.owner,
            "timezone": self.timezone,
            "is_active": self.is_active,
            "is_current": self.is_current,
        }

    def add_object(
        self,
        name: str,
        type: str | None = None,
        description: str | None = None,
        is_active: bool = True,
        is_current: bool = True,
    ) -> Object:
        """Create and add a new Object to this Source.

        Raises ValueError if an object with this name already exists.
        """
        if name in self.objects:
            raise ValueError(
                f"An object with name '{name}' already exists in source '{self.name}'."
            )

        obj = Object(
            name=name,
            source=self,
            type=type,
            description=description,
            is_active=is_active,
            is_current=is_current,
        )
        self.objects[name] = obj
        self.metadata._client.write_table("object", [obj.to_dict()])
        return obj

    def get_object(self, name: str) -> Object | None:
        """Retrieve an Object by name from this Source."""
        return self.objects.get(name)

    def get_objects(self) -> list[Object]:
        """Return all Objects in this Source."""
        return list(self.objects.values())

    def get_active_objects(self) -> list[Object]:
        """Return all active Objects in this Source."""
        return [obj for obj in self.objects.values() if obj.is_active]

    def get_current_objects(self) -> list[Object]:
        """Return all current Objects in this Source."""
        return [obj for obj in self.objects.values() if obj.is_current]

    def update(self, **fields: Any) -> bool:
        """Update metadata for this source.

        Returns True if updated, False if not found.
        Raises ValueError for unknown or non-updatable columns.
        """
        return self.metadata._update_source(self.name, **fields)

    def delete(self) -> None:
        """Mark this source as inactive and not current."""
        self.is_active = False
        self.is_current = False
        self.update(is_active=False, is_current=False)

    def _load_objects_from_db(self) -> None:
        """Load all objects for this source from the database."""
        rows = self.metadata._client.read_table("object")
        self.objects.clear()
        for row in rows:
            if row["source_name"] == self.name:
                obj = Object(
                    name=row["name"],
                    source=self,
                    type=row.get("type"),
                    description=row.get("description"),
                    is_active=bool(row["is_active"]),
                    is_current=bool(row["is_current"]),
                )
                self.objects[obj.name] = obj


# ---------------------------------------------------------------------------
# Metadata (root node)
# ---------------------------------------------------------------------------


class Metadata:
    """Top-level container for the metadata database and all sources/objects.

    The Metadata class manages:
    - Connection to the metadata database via a DatabaseClient
    - Creation and initialization of metadata tables
    - Access to all Sources and their Objects
    - Hierarchical navigation of the metadata tree

    Example:
        >>> metadata = Metadata(client)
        >>> metadata.create_tables()
        >>> source = metadata.add_source(name="crm", type="database")
        >>> obj = source.add_object(name="customers", type="table")
        >>> print(obj.source.name)  # "crm"
        >>> print(obj.source.metadata)  # <Metadata object>
    """

    def __init__(self, client: DatabaseClient) -> None:
        """Initialize Metadata with a database client."""
        self._client = client
        self.sources: dict[str, Source] = {}

    def create_tables(self, if_not_exists: bool = True) -> tuple[str, str]:
        """Create metadata tables and return the executed SQL statements.

        Creates both the 'source' and 'object' tables.

        Args:
            if_not_exists: If True, use CREATE TABLE IF NOT EXISTS.

        Returns:
            A tuple of (source_sql, object_sql) statements.
        """
        from .object_repository import OBJECT_TABLE
        from .source_repository import SOURCE_TABLE
        from .table_creator import MetadataTableCreator

        creator = MetadataTableCreator(self._client)
        source_sql = creator.create_table(SOURCE_TABLE, if_not_exists=if_not_exists)
        object_sql = creator.create_table(OBJECT_TABLE, if_not_exists=if_not_exists)
        return source_sql, object_sql

    def add_source(
        self,
        name: str,
        type: str | None = None,
        system: str | None = None,
        owner: str | None = None,
        timezone: str | None = None,
        is_active: bool = True,
        is_current: bool = True,
    ) -> Source:
        """Create and add a new Source to this Metadata.

        Raises ValueError if a source with this name already exists.
        """
        if name in self.sources:
            raise ValueError(f"A source with name '{name}' already exists.")

        source = Source(
            name=name,
            metadata=self,
            type=type,
            system=system,
            owner=owner,
            timezone=timezone,
            is_active=is_active,
            is_current=is_current,
        )
        self.sources[name] = source
        self._client.write_table("source", [source.to_dict()])
        return source

    def get_source(self, name: str) -> Source | None:
        """Retrieve a Source by name."""
        return self.sources.get(name)

    def get_sources(self) -> list[Source]:
        """Return all Sources."""
        return list(self.sources.values())

    def get_active_sources(self) -> list[Source]:
        """Return all active Sources."""
        return [source for source in self.sources.values() if source.is_active]

    def get_current_sources(self) -> list[Source]:
        """Return all current Sources."""
        return [source for source in self.sources.values() if source.is_current]

    def get_all_objects(self) -> list[Object]:
        """Return all Objects across all Sources."""
        all_objects: list[Object] = []
        for source in self.sources.values():
            all_objects.extend(source.get_objects())
        return all_objects

    def load_from_database(self) -> None:
        """Load the complete metadata tree from the database.

        This method should be called after construction to populate the tree
        with existing metadata from the database tables.
        """
        self.sources.clear()

        # Load all sources
        source_rows = self._client.read_table("source")
        for row in source_rows:
            source = Source(
                name=row["name"],
                metadata=self,
                type=row.get("type"),
                system=row.get("system"),
                owner=row.get("owner"),
                timezone=row.get("timezone"),
                is_active=bool(row["is_active"]),
                is_current=bool(row["is_current"]),
            )
            self.sources[source.name] = source

        # Load all objects for each source
        for source in self.sources.values():
            source._load_objects_from_db()

    def _update_source(self, name: str, **fields: Any) -> bool:
        """Internal: Update a source record in the database.

        Returns True if updated, False if not found.
        """
        allowed_fields = {"type", "system", "owner", "timezone", "is_active", "is_current"}
        unknown = set(fields) - allowed_fields
        if unknown:
            raise ValueError(
                f"Unknown or non-updatable field(s): {sorted(unknown)}. "
                f"Allowed: {sorted(allowed_fields)}"
            )
        if not fields:
            return False

        if name not in self.sources:
            return False

        set_clause = ", ".join(f"{col} = ?" for col in fields)
        params: list[Any] = list(fields.values()) + [name]
        self._client.execute(
            f"UPDATE source SET {set_clause} WHERE name = ?",  # noqa: S608
            params=params,
        )

        # Update the in-memory object
        source = self.sources[name]
        for key, value in fields.items():
            setattr(source, key, value)

        return True

    def _update_object(self, name: str, **fields: Any) -> bool:
        """Internal: Update an object record in the database.

        Returns True if updated, False if not found.
        """
        allowed_fields = {"type", "description", "is_active", "is_current"}
        unknown = set(fields) - allowed_fields
        if unknown:
            raise ValueError(
                f"Unknown or non-updatable field(s): {sorted(unknown)}. "
                f"Allowed: {sorted(allowed_fields)}"
            )
        if not fields:
            return False

        # Find the object across all sources
        target_obj = None
        for source in self.sources.values():
            obj = source.get_object(name)
            if obj is not None:
                target_obj = obj
                break

        if target_obj is None:
            return False

        set_clause = ", ".join(f"{col} = ?" for col in fields)
        params: list[Any] = list(fields.values()) + [name]
        self._client.execute(
            f"UPDATE object SET {set_clause} WHERE name = ?",  # noqa: S608
            params=params,
        )

        # Update the in-memory object
        for key, value in fields.items():
            setattr(target_obj, key, value)

        return True
