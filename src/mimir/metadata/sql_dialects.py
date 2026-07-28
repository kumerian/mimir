"""SQL dialect adapters that compile logical metadata schemas to DDL."""

from __future__ import annotations

from abc import ABC, abstractmethod

from .schema import ColumnDefinition, TableDefinition


class SQLDialect(ABC):
    """Base SQL dialect API for table creation statements."""

    identifier_quote_start = '"'
    identifier_quote_end = '"'

    def create_table_sql(self, table: TableDefinition, if_not_exists: bool = True) -> str:
        """Build a CREATE TABLE statement for the provided logical table."""

        self._validate_table(table)

        table_name = self.quote_identifier_path(table.name)
        if_not_exists_sql = self._if_not_exists_clause(if_not_exists)

        column_sql = ", ".join(self._column_sql(column) for column in table.columns)

        return f"CREATE TABLE {if_not_exists_sql}{table_name} ({column_sql})"

    def _column_sql(self, column: ColumnDefinition) -> str:
        parts = [
            self.quote_identifier_path(column.name),
            self._type_sql(column),
        ]

        if not column.nullable:
            parts.append("NOT NULL")

        if column.default is not None:
            parts.append(f"DEFAULT {column.default}")

        if column.primary_key:
            parts.append("PRIMARY KEY")

        return " ".join(parts)

    @abstractmethod
    def _if_not_exists_clause(self, if_not_exists: bool) -> str:
        """Return the dialect-specific IF NOT EXISTS clause fragment."""

    @abstractmethod
    def _type_sql(self, column: ColumnDefinition) -> str:
        """Map a logical column type to a physical database type."""

    def quote_identifier_path(self, identifier_path: str) -> str:
        """Quote each identifier path segment (for example schema.table)."""

        return ".".join(self._quote_identifier(part) for part in identifier_path.split("."))

    def _quote_identifier(self, identifier: str) -> str:
        return (
            f"{self.identifier_quote_start}{identifier}{self.identifier_quote_end}"
        )

    def _validate_table(self, table: TableDefinition) -> None:
        if not table.columns:
            raise ValueError("table must define at least one column")


class StandardIfNotExistsDialect(SQLDialect):
    """Dialect for engines that support IF NOT EXISTS natively."""

    def _if_not_exists_clause(self, if_not_exists: bool) -> str:
        return "IF NOT EXISTS " if if_not_exists else ""


class DuckDBDialect(StandardIfNotExistsDialect):
    def _type_sql(self, column: ColumnDefinition) -> str:
        return _default_type_sql(column)


class PostgreSQLDialect(StandardIfNotExistsDialect):
    def _type_sql(self, column: ColumnDefinition) -> str:
        return _default_type_sql(column)


class MySQLDialect(StandardIfNotExistsDialect):
    identifier_quote_start = "`"
    identifier_quote_end = "`"

    def _type_sql(self, column: ColumnDefinition) -> str:
        return _mysql_type_sql(column)


class MariaDBDialect(MySQLDialect):
    pass


class MSSQLDialect(SQLDialect):
    identifier_quote_start = "["
    identifier_quote_end = "]"

    def _if_not_exists_clause(self, if_not_exists: bool) -> str:
        if if_not_exists:
            raise ValueError("if_not_exists is not supported for MSSQL create table statements")
        return ""

    def _type_sql(self, column: ColumnDefinition) -> str:
        return _mssql_type_sql(column)


class OracleDialect(SQLDialect):
    def _if_not_exists_clause(self, if_not_exists: bool) -> str:
        if if_not_exists:
            raise ValueError("if_not_exists is not supported for Oracle create table statements")
        return ""

    def _type_sql(self, column: ColumnDefinition) -> str:
        return _oracle_type_sql(column)


def _default_type_sql(column: ColumnDefinition) -> str:
    if column.column_type == "string":
        length = column.length or 255
        return f"VARCHAR({length})"
    if column.column_type == "text":
        return "TEXT"
    if column.column_type == "integer":
        return "INTEGER"
    if column.column_type == "bigint":
        return "BIGINT"
    if column.column_type == "float":
        return "DOUBLE"
    if column.column_type == "boolean":
        return "BOOLEAN"
    if column.column_type == "date":
        return "DATE"
    if column.column_type == "timestamp":
        return "TIMESTAMP"
    if column.column_type == "decimal":
        precision = column.precision or 18
        scale = column.scale or 2
        return f"DECIMAL({precision}, {scale})"
    raise ValueError(f"Unsupported column_type: {column.column_type}")


def _mysql_type_sql(column: ColumnDefinition) -> str:
    if column.column_type == "float":
        return "DOUBLE"
    return _default_type_sql(column)


def _mssql_type_sql(column: ColumnDefinition) -> str:
    if column.column_type == "string":
        length = column.length or 255
        return f"NVARCHAR({length})"
    if column.column_type == "text":
        return "NVARCHAR(MAX)"
    if column.column_type == "integer":
        return "INT"
    if column.column_type == "bigint":
        return "BIGINT"
    if column.column_type == "float":
        return "FLOAT"
    if column.column_type == "boolean":
        return "BIT"
    if column.column_type == "date":
        return "DATE"
    if column.column_type == "timestamp":
        return "DATETIME2"
    if column.column_type == "decimal":
        precision = column.precision or 18
        scale = column.scale or 2
        return f"DECIMAL({precision}, {scale})"
    raise ValueError(f"Unsupported column_type: {column.column_type}")


def _oracle_type_sql(column: ColumnDefinition) -> str:
    if column.column_type == "string":
        length = column.length or 255
        return f"VARCHAR2({length})"
    if column.column_type == "text":
        return "CLOB"
    if column.column_type == "integer":
        return "NUMBER(10, 0)"
    if column.column_type == "bigint":
        return "NUMBER(19, 0)"
    if column.column_type == "float":
        return "BINARY_DOUBLE"
    if column.column_type == "boolean":
        return "NUMBER(1, 0)"
    if column.column_type == "date":
        return "DATE"
    if column.column_type == "timestamp":
        return "TIMESTAMP"
    if column.column_type == "decimal":
        precision = column.precision or 18
        scale = column.scale or 2
        return f"NUMBER({precision}, {scale})"
    raise ValueError(f"Unsupported column_type: {column.column_type}")
