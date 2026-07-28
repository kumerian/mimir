"""High-level metadata service for creating tables across database engines."""

from __future__ import annotations

from mimir.utility import (
    DatabaseClient,
    DuckDBClient,
    MariaDBClient,
    MSSQLClient,
    MySQLClient,
    OracleClient,
    PostgreSQLClient,
)

from .schema import TableDefinition
from .sql_dialects import (
    DuckDBDialect,
    MSSQLDialect,
    MariaDBDialect,
    MySQLDialect,
    OracleDialect,
    PostgreSQLDialect,
    SQLDialect,
)


class MetadataTableCreator:
    """Create tables from logical metadata definitions in a database-agnostic way."""

    def __init__(self, client: DatabaseClient) -> None:
        self._client = client
        self._dialect = _resolve_dialect(client)

    def build_create_table_sql(
        self,
        table: TableDefinition,
        if_not_exists: bool = True,
    ) -> str:
        """Build the CREATE TABLE SQL statement for the client's database engine."""

        return self._dialect.create_table_sql(table=table, if_not_exists=if_not_exists)

    def create_table(
        self,
        table: TableDefinition,
        if_not_exists: bool = True,
    ) -> str:
        """Create a table in the connected database and return the executed SQL."""

        statement = self.build_create_table_sql(table=table, if_not_exists=if_not_exists)
        self._client.execute(statement)
        return statement


def _resolve_dialect(client: DatabaseClient) -> SQLDialect:
    if isinstance(client, DuckDBClient):
        return DuckDBDialect()
    if isinstance(client, PostgreSQLClient):
        return PostgreSQLDialect()
    if isinstance(client, MySQLClient):
        return MySQLDialect()
    if isinstance(client, MariaDBClient):
        return MariaDBDialect()
    if isinstance(client, MSSQLClient):
        return MSSQLDialect()
    if isinstance(client, OracleClient):
        return OracleDialect()

    raise ValueError(
        f"Unsupported DatabaseClient type for metadata table creation: {type(client).__name__}"
    )
