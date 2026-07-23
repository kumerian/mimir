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

__all__ = [
    "DatabaseClient",
    "DuckDBClient",
    "MSSQLClient",
    "PostgreSQLClient",
    "MariaDBClient",
    "MySQLClient",
    "OracleClient",
]
__version__ = "0.1.0"
