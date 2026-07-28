from __future__ import annotations

from typing import Any

import pytest

from mimir.metadata import ColumnDefinition, MetadataTableCreator, TableDefinition
from mimir.utility import (
    DatabaseClient,
    DuckDBClient,
    MariaDBClient,
    MSSQLClient,
    MySQLClient,
    OracleClient,
    PostgreSQLClient,
)


class _FakeCursor:
    def __init__(self) -> None:
        self.executed: list[tuple[str, tuple[Any, ...] | None]] = []

    def execute(self, query: str, params: tuple[Any, ...] | None = None) -> None:
        self.executed.append((query, params))


class _FakeConnection:
    def __init__(self) -> None:
        self.cursor_instance = _FakeCursor()
        self.committed = False

    def cursor(self) -> _FakeCursor:
        return self.cursor_instance

    def commit(self) -> None:
        self.committed = True


class _FakeDuckDBClient(DuckDBClient):
    def connect(self, auth_method: str = "path", **kwargs: Any) -> None:
        _ = auth_method
        _ = kwargs
        self._connection = _FakeConnection()

    def get_test_connection(self) -> _FakeConnection:
        if self._connection is None:
            raise RuntimeError("Expected test connection to be initialized")
        return self._connection


def _build_table() -> TableDefinition:
    return TableDefinition(
        name="metadata.records",
        columns=(
            ColumnDefinition(name="id", column_type="integer", nullable=False, primary_key=True),
            ColumnDefinition(name="title", column_type="string", length=120, nullable=False),
            ColumnDefinition(name="score", column_type="decimal", precision=12, scale=4),
            ColumnDefinition(name="created_at", column_type="timestamp"),
        ),
    )


@pytest.mark.unit
@pytest.mark.parametrize(
    ("client", "if_not_exists", "expected_sql"),
    [
        (
            DuckDBClient(),
            True,
            "CREATE TABLE IF NOT EXISTS \"metadata\".\"records\" "
            "(\"id\" INTEGER NOT NULL PRIMARY KEY, \"title\" VARCHAR(120) NOT NULL, "
            "\"score\" DECIMAL(12, 4), \"created_at\" TIMESTAMP)",
        ),
        (
            PostgreSQLClient(),
            True,
            "CREATE TABLE IF NOT EXISTS \"metadata\".\"records\" "
            "(\"id\" INTEGER NOT NULL PRIMARY KEY, \"title\" VARCHAR(120) NOT NULL, "
            "\"score\" DECIMAL(12, 4), \"created_at\" TIMESTAMP)",
        ),
        (
            MySQLClient(),
            True,
            "CREATE TABLE IF NOT EXISTS `metadata`.`records` "
            "(`id` INTEGER NOT NULL PRIMARY KEY, `title` VARCHAR(120) NOT NULL, "
            "`score` DECIMAL(12, 4), `created_at` TIMESTAMP)",
        ),
        (
            MariaDBClient(),
            True,
            "CREATE TABLE IF NOT EXISTS `metadata`.`records` "
            "(`id` INTEGER NOT NULL PRIMARY KEY, `title` VARCHAR(120) NOT NULL, "
            "`score` DECIMAL(12, 4), `created_at` TIMESTAMP)",
        ),
        (
            MSSQLClient(),
            False,
            "CREATE TABLE [metadata].[records] "
            "([id] INT NOT NULL PRIMARY KEY, [title] NVARCHAR(120) NOT NULL, "
            "[score] DECIMAL(12, 4), [created_at] DATETIME2)",
        ),
        (
            OracleClient(),
            False,
            "CREATE TABLE \"metadata\".\"records\" "
            "(\"id\" NUMBER(10, 0) NOT NULL PRIMARY KEY, \"title\" VARCHAR2(120) NOT NULL, "
            "\"score\" NUMBER(12, 4), \"created_at\" TIMESTAMP)",
        ),
    ],
)
def test_build_create_table_sql_for_supported_clients(
    client: DatabaseClient,
    if_not_exists: bool,
    expected_sql: str,
) -> None:
    creator = MetadataTableCreator(client)

    sql = creator.build_create_table_sql(_build_table(), if_not_exists=if_not_exists)

    assert sql == expected_sql


@pytest.mark.unit
def test_create_table_executes_sql_statement() -> None:
    client = _FakeDuckDBClient()
    client.connect()

    creator = MetadataTableCreator(client)
    sql = creator.create_table(_build_table())

    connection = client.get_test_connection()
    assert connection.cursor_instance.executed == [(sql, None)]
    assert connection.committed is True


@pytest.mark.unit
def test_if_not_exists_not_supported_for_mssql() -> None:
    creator = MetadataTableCreator(MSSQLClient())

    with pytest.raises(ValueError, match="if_not_exists is not supported"):
        creator.build_create_table_sql(_build_table(), if_not_exists=True)
