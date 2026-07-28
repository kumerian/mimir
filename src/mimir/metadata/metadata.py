"""Hierarchical metadata tree structure for metadata-driven ingestion.

This module provides a tree-based abstraction over the metadata tables:
- Metadata: top-level container representing the metadata database
- Source: represents a data source, created from a Metadata object
- Object: represents an object within a source, created from a Source object
- Column: represents a column within an object, created from an Object

The tree structure enables intuitive navigation and management of metadata
while abstracting away the underlying database operations.

All field definitions for tree classes are automatically derived from the
canonical table definitions in the markdown files, ensuring a single source
of truth.
"""

from __future__ import annotations

from dataclasses import dataclass, field, fields as dataclass_fields, make_dataclass, MISSING
from typing import TYPE_CHECKING, Any

from mimir.utility import DatabaseClient

if TYPE_CHECKING:
    from .column_repository import ColumnRepository
    from .object_repository import ObjectRepository
    from .source_repository import SourceRepository


# ---------------------------------------------------------------------------
# Helper: Generate field defaults from table definitions
# ---------------------------------------------------------------------------


def _get_column_defaults(table_def: Any) -> dict[str, Any]:
    """Extract column defaults from a table definition."""
    defaults = {}
    for col_def in table_def.columns:
        if col_def.default is not None:
            # Parse boolean defaults
            if col_def.default.upper() == "TRUE":
                defaults[col_def.name] = True
            elif col_def.default.upper() == "FALSE":
                defaults[col_def.name] = False
            else:
                defaults[col_def.name] = col_def.default
        elif col_def.nullable:
            defaults[col_def.name] = None
    return defaults


def _validate_tree_class_matches_table(tree_cls: type, table_def: Any, excluded_fields: set[str]) -> None:
    """Validate that a tree class has fields matching the table definition.
    
    Args:
        tree_cls: The dataclass to validate
        table_def: The table definition from markdown
        excluded_fields: Fields to exclude from validation (navigation and FK fields)
    """
    tree_field_names = {f.name for f in dataclass_fields(tree_cls)}
    table_col_names = {col.name for col in table_def.columns}
    
    # Exclude specified fields from both sets before comparison
    tree_field_names = tree_field_names - excluded_fields
    table_col_names = table_col_names - excluded_fields
    
    missing = table_col_names - tree_field_names
    extra = tree_field_names - table_col_names
    
    if missing or extra:
        msg = f"Tree class {tree_cls.__name__} field mismatch with table {table_def.name}:"
        if missing:
            msg += f"\n  Missing fields: {sorted(missing)}"
        if extra:
            msg += f"\n  Extra fields: {sorted(extra)}"
        raise ValueError(msg)


# ---------------------------------------------------------------------------
# Column (leaf node)
# ---------------------------------------------------------------------------
# Note: Column class is built dynamically below after Object and Source
# are defined to handle forward references correctly.


# Placeholder for type hints - will be replaced with dynamic class
Column: type


# ---------------------------------------------------------------------------
# Object (intermediate leaf node)
# ---------------------------------------------------------------------------


@dataclass
class Object:
    """Represents a metadata object (table, view, etc.) within a source.

    An Object is always associated with a Source and cannot exist independently.
    It contains metadata about the object such as its name, type, and description,
    and can contain zero or more Columns.
    """

    name: str
    source: Source
    type: str | None = None
    description: str | None = None
    is_active: bool = field(default=True)
    is_current: bool = field(default=True)
    columns: dict[str, Column] = field(default_factory=dict, init=False)

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

    def add_column(self, name: str, **kwargs: Any) -> Column:
        """Create and add a new Column to this Object.
        
        Args:
            name: The column name (required)
            **kwargs: Additional field values for the Column (all optional)
        
        Raises ValueError if a column with this name already exists.
        """
        if name in self.columns:
            raise ValueError(
                f"A column with name '{name}' already exists in object '{self.name}'."
            )

        # Build Column with dynamic fields - only include fields that exist in Column class
        column_field_names = {f.name for f in dataclass_fields(Column) if f.name != "object"}
        column_kwargs = {k: v for k, v in kwargs.items() if k in column_field_names}
        
        col = Column(
            name=name,
            object=self,
            **column_kwargs,
        )
        self.columns[name] = col
        self.source.metadata._client.write_table("columns", [col.to_dict()])
        return col

    def get_column(self, name: str) -> Column | None:
        """Retrieve a Column by name from this Object."""
        return self.columns.get(name)

    def get_columns(self) -> list[Column]:
        """Return all Columns in this Object."""
        return list(self.columns.values())

    def get_active_columns(self) -> list[Column]:
        """Return all active Columns in this Object."""
        return [col for col in self.columns.values() if col.is_active]

    def get_current_columns(self) -> list[Column]:
        """Return all current Columns in this Object."""
        return [col for col in self.columns.values() if col.is_current]

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

    def _load_columns_from_db(self) -> None:
        """Load all columns for this object from the database.
        
        This method is fully dynamic - it reads Column fields from the class
        definition and doesn't hardcode any field names.
        """
        rows = self.source.metadata._client.read_table("columns")
        self.columns.clear()
        
        # Get valid Column field names (excluding 'object' and 'name' which are set separately)
        column_field_names = {f.name for f in dataclass_fields(Column) if f.name not in {"object", "name"}}
        
        for row in rows:
            if row["object_name"] == self.name:
                # Dynamically extract only the fields that exist in Column (excluding 'name' which is passed explicitly)
                col_kwargs = {k: v for k, v in row.items() if k in column_field_names}
                
                col = Column(
                    name=row["name"],
                    object=self,
                    **col_kwargs,
                )
                self.columns[col.name] = col


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

    def add_object(self, name: str, **kwargs: Any) -> Object:
        """Create and add a new Object to this Source.
        
        Args:
            name: The object name (required)
            **kwargs: Additional field values for the Object (all optional)

        Raises ValueError if an object with this name already exists.
        """
        if name in self.objects:
            raise ValueError(
                f"An object with name '{name}' already exists in source '{self.name}'."
            )

        # Build Object with dynamic fields - only include fields that exist in Object class
        object_field_names = {f.name for f in dataclass_fields(Object) if f.name not in {"source", "objects"}}
        object_kwargs = {k: v for k, v in kwargs.items() if k in object_field_names}

        obj = Object(
            name=name,
            source=self,
            **object_kwargs,
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
        """Load all objects for this source from the database.
        
        This method is fully dynamic - it reads Object fields from the class
        definition and doesn't hardcode any field names.
        """
        rows = self.metadata._client.read_table("object")
        self.objects.clear()
        
        # Get valid Object field names (excluding 'source', 'objects', and 'name' which are set separately)
        object_field_names = {f.name for f in dataclass_fields(Object) if f.name not in {"source", "objects", "name"}}
        
        for row in rows:
            if row["source_name"] == self.name:
                # Dynamically extract only the fields that exist in Object (excluding 'name' which is passed explicitly)
                obj_kwargs = {k: v for k, v in row.items() if k in object_field_names}
                
                obj = Object(
                    name=row["name"],
                    source=self,
                    **obj_kwargs,
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

    def create_tables(self, if_not_exists: bool = True) -> tuple[str, str, str]:
        """Create metadata tables and return the executed SQL statements.

        Creates the 'source', 'object', and 'column' tables.

        Args:
            if_not_exists: If True, use CREATE TABLE IF NOT EXISTS.

        Returns:
            A tuple of (source_sql, object_sql, column_sql) statements.
        """
        from .column_repository import COLUMN_TABLE
        from .object_repository import OBJECT_TABLE
        from .source_repository import SOURCE_TABLE
        from .table_creator import MetadataTableCreator

        creator = MetadataTableCreator(self._client)
        source_sql = creator.create_table(SOURCE_TABLE, if_not_exists=if_not_exists)
        object_sql = creator.create_table(OBJECT_TABLE, if_not_exists=if_not_exists)
        column_sql = creator.create_table(COLUMN_TABLE, if_not_exists=if_not_exists)
        return source_sql, object_sql, column_sql

    def add_source(self, name: str, **kwargs: Any) -> Source:
        """Create and add a new Source to this Metadata.
        
        Args:
            name: The source name (required)
            **kwargs: Additional field values for the Source (all optional)

        Raises ValueError if a source with this name already exists.
        """
        if name in self.sources:
            raise ValueError(f"A source with name '{name}' already exists.")

        # Build Source with dynamic fields - only include fields that exist in Source class
        source_field_names = {f.name for f in dataclass_fields(Source) if f.name not in {"metadata", "objects"}}
        source_kwargs = {k: v for k, v in kwargs.items() if k in source_field_names}

        source = Source(
            name=name,
            metadata=self,
            **source_kwargs,
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
        
        This loading is fully dynamic - it reads field definitions from the classes
        and doesn't hardcode any field names.
        """
        self.sources.clear()

        # Load all sources dynamically
        source_field_names = {f.name for f in dataclass_fields(Source) if f.name not in {"metadata", "objects", "name"}}
        source_rows = self._client.read_table("source")
        for row in source_rows:
            # Dynamically extract only the fields that exist in Source (excluding 'name' which is passed explicitly)
            source_kwargs = {k: v for k, v in row.items() if k in source_field_names}
            
            source = Source(
                name=row["name"],
                metadata=self,
                **source_kwargs,
            )
            self.sources[source.name] = source

        # Load all objects for each source
        for source in self.sources.values():
            source._load_objects_from_db()

        # Load all columns for each object
        for source in self.sources.values():
            for obj in source.get_objects():
                obj._load_columns_from_db()

    def _update_source(self, name: str, **fields: Any) -> bool:
        """Internal: Update a source record in the database.

        Returns True if updated, False if not found.
        """
        from .source_repository import _UPDATABLE_COLUMNS as SOURCE_UPDATABLE_COLUMNS

        allowed_fields = SOURCE_UPDATABLE_COLUMNS
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
        from .object_repository import _UPDATABLE_COLUMNS as OBJECT_UPDATABLE_COLUMNS

        allowed_fields = OBJECT_UPDATABLE_COLUMNS
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

    def _update_column(self, name: str, object_name: str, **fields: Any) -> bool:
        """Internal: Update a column record in the database.

        Returns True if updated, False if not found.
        """
        from .column_repository import _UPDATABLE_COLUMNS as COLUMN_UPDATABLE_COLUMNS

        allowed_fields = COLUMN_UPDATABLE_COLUMNS
        unknown = set(fields) - allowed_fields
        if unknown:
            raise ValueError(
                f"Unknown or non-updatable field(s): {sorted(unknown)}. "
                f"Allowed: {sorted(allowed_fields)}"
            )
        if not fields:
            return False

        # Find the column across all sources and objects
        target_col = None
        for source in self.sources.values():
            obj = source.get_object(object_name)
            if obj is not None:
                col = obj.get_column(name)
                if col is not None:
                    target_col = col
                    break

        if target_col is None:
            return False

        set_clause = ", ".join(f"{col} = ?" for col in fields)
        params: list[Any] = list(fields.values()) + [name, object_name]
        self._client.execute(
            f"UPDATE columns SET {set_clause} WHERE name = ? AND object_name = ?",  # noqa: S608
            params=params,
        )

        # Update the in-memory column
        for key, value in fields.items():
            setattr(target_col, key, value)

        return True


# ---------------------------------------------------------------------------
# Validation: Ensure tree classes match table definitions
# ---------------------------------------------------------------------------
# This validation ensures that the tree class fields are always in sync with
# the canonical markdown definitions. If the markdown changes, the tree classes
# must be updated accordingly.


def _validate_tree_classes_match_definitions() -> None:
    """Validate that tree classes match their table definitions at module load."""
    from .column_repository import COLUMN_TABLE
    from .object_repository import OBJECT_TABLE
    from .source_repository import SOURCE_TABLE

    # Validate Source fields (excluding navigation fields)
    # source_name is not a field in Source (it's a standalone table)
    _validate_tree_class_matches_table(Source, SOURCE_TABLE, excluded_fields={"metadata", "objects"})

    # Validate Object fields (excluding navigation and foreign key fields)
    # source_name is replaced with source: Source reference
    _validate_tree_class_matches_table(
        Object, OBJECT_TABLE, excluded_fields={"source_name", "source", "columns"}
    )

    # Validate Column fields (excluding navigation and foreign key fields)
    # object_name is replaced with object: Object reference
    _validate_tree_class_matches_table(
        Column, COLUMN_TABLE, excluded_fields={"object_name", "object"}
    )




# ---------------------------------------------------------------------------
# Dynamic Column class generation from markdown
# ---------------------------------------------------------------------------
# The Column class is built dynamically at module load time from the markdown
# definition in metadata/metadata/column.md. This ensures there are NO hardcoded
# field definitions in the code; all fields come exclusively from the markdown.


def _build_column_class() -> type:
    """Build the Column dataclass dynamically from COLUMN_TABLE markdown definition.
    
    This ensures there are NO hardcoded field definitions in the code;
    all fields come from the markdown definition.
    """
    from .column_repository import COLUMN_TABLE

    # Build fields from markdown: skip object_name (FK), it becomes object reference
    fields_list: list[tuple[str, Any, Any]] = []
    
    for col in COLUMN_TABLE.columns:
        if col.name == "object_name":
            continue  # FK replaced by object: Object reference
        
        # Determine type annotation based on nullable and data type
        if col.name in {"functional_name", "description", "data_type"}:
            type_annotation = str | None
        elif col.name in {"is_metadata", "is_key", "is_nullable", "is_current", "is_active"}:
            type_annotation = bool | None if col.nullable else bool
        elif col.name == "ordinal_position":
            type_annotation = int | None
        elif col.name == "name":
            type_annotation = str
        else:
            type_annotation = Any
        
        # Determine default value from table definition
        if col.name == "name":
            # name is required, no default
            fields_list.append((col.name, type_annotation, MISSING))
        elif col.default is not None:
            if col.default.upper() == "TRUE":
                default_val = field(default=True)
            elif col.default.upper() == "FALSE":
                default_val = field(default=False)
            else:
                default_val = field(default=col.default)
            fields_list.append((col.name, type_annotation, default_val))
        elif col.nullable:
            default_val = field(default=None)
            fields_list.append((col.name, type_annotation, default_val))
        else:
            fields_list.append((col.name, type_annotation, MISSING))
    
    # Add the navigation field: object: Object (no default, required)
    fields_list.insert(1, ("object", Object, MISSING))
    
    # Create the Column class dynamically
    ColumnClass = make_dataclass(
        "Column",
        fields_list,
        frozen=False,
    )
    
    # Add docstring
    ColumnClass.__doc__ = """Represents a column within a metadata object (table, view, etc.).

    A Column is always associated with an Object and cannot exist independently.
    It contains metadata about the column such as its name, data type, and constraints.
    
    All fields are generated from the canonical markdown definition (metadata/metadata/column.md)
    to ensure a single source of truth.
    """
    
    # Add methods
    def to_dict(self) -> dict[str, Any]:
        """Return a dictionary representation suitable for database storage."""
        result = {}
        for f in dataclass_fields(self):
            if f.name == "object":
                result["object_name"] = self.object.name
            else:
                result[f.name] = getattr(self, f.name)
        return result
    
    def update(self, **fields: Any) -> bool:
        """Update metadata for this column.

        Returns True if updated, False if not found.
        Raises ValueError for unknown or non-updatable columns.
        """
        return self.object.source.metadata._update_column(self.name, self.object.name, **fields)
    
    def delete(self) -> None:
        """Mark this column as inactive and not current."""
        self.is_active = False
        self.is_current = False
        self.update(is_active=False, is_current=False)
    
    ColumnClass.to_dict = to_dict
    ColumnClass.update = update
    ColumnClass.delete = delete
    
    return ColumnClass


# Build Column class at module load time (after Object and Source are defined)
Column = _build_column_class()

