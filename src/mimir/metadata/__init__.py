"""Metadata primitives for database-agnostic table management."""

from .schema import ColumnDefinition, ColumnType, TableDefinition
from .source_repository import SOURCE_TABLE, SourceRecord, SourceRepository
from .table_creator import MetadataTableCreator

__all__ = [
    "ColumnDefinition",
    "ColumnType",
    "TableDefinition",
    "MetadataTableCreator",
    "SOURCE_TABLE",
    "SourceRecord",
    "SourceRepository",
]
