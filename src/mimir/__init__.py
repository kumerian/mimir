"""Public package interface for the Mimir library."""

from .utility import (
    DatabaseClient,
    DuckDBClient,
    MariaDBClient,
    MSSQLClient,
    MySQLClient,
    OracleClient,
    PostgreSQLClient,
)
from .metadata import ColumnDefinition, ColumnType, MetadataTableCreator, TableDefinition

__all__ = [
    "DatabaseClient",
    "DuckDBClient",
    "MSSQLClient",
    "PostgreSQLClient",
    "MariaDBClient",
    "MySQLClient",
    "OracleClient",
    "ColumnType",
    "ColumnDefinition",
    "TableDefinition",
    "MetadataTableCreator",
]
__version__ = "0.1.0"
