"""Metadata primitives for database-agnostic table management."""

from .column_repository import COLUMN_TABLE, ColumnRecord, ColumnRepository
from .metadata import Column, Metadata, Object, Source
from .object_repository import OBJECT_TABLE, ObjectRecord, ObjectRepository
from .schema import ColumnDefinition, ColumnType, TableDefinition
from .source_repository import SOURCE_TABLE, SourceRecord, SourceRepository
from .table_creator import MetadataTableCreator

__all__ = [
    # Hierarchical tree structure (new)
    "Metadata",
    "Source",
    "Object",
    "Column",
    # Schema definitions
    "ColumnDefinition",
    "ColumnType",
    "TableDefinition",
    "MetadataTableCreator",
    # Legacy repositories (deprecated)
    "COLUMN_TABLE",
    "ColumnRecord",
    "ColumnRepository",
    "OBJECT_TABLE",
    "ObjectRecord",
    "ObjectRepository",
    "SOURCE_TABLE",
    "SourceRecord",
    "SourceRepository",
]
