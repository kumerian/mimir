"""Shared database utility abstractions for Mimir."""

from __future__ import annotations

import re
from abc import ABC, abstractmethod
from typing import Any, Mapping, Sequence


class DatabaseClient(ABC):
    """Base class for database utility implementations.

    Instances are created without an active connection. Call ``connect`` before
    using ``read_table`` or ``write_table``.
    """

    _IDENTIFIER_PATTERN = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*$")

    def __init__(self) -> None:
        self._connection: Any | None = None

    @property
    def is_connected(self) -> bool:
        """Return ``True`` when a connection object is available."""

        return self._connection is not None

    @abstractmethod
    def connect(self, auth_method: str = "password", **kwargs: Any) -> None:
        """Connect to the target database using a supported authentication mode."""

    def disconnect(self) -> None:
        """Close the active connection if one exists."""

        if self._connection is not None:
            self._connection.close()
            self._connection = None

    def read_table(
        self,
        table_name: str,
        columns: Sequence[str] | None = None,
        limit: int | None = None,
    ) -> list[dict[str, Any]]:
        """Read table rows and return them as a list of dictionaries."""

        connection = self._require_connection()
        table = self._validate_identifier(table_name, "table_name")

        if columns:
            selected_columns = ", ".join(self._validate_identifier(c, "column") for c in columns)
        else:
            selected_columns = "*"

        query = f"SELECT {selected_columns} FROM {table}"
        if limit is not None:
            if limit <= 0:
                raise ValueError("limit must be greater than 0")
            query = f"{query} LIMIT {limit}"

        cursor = connection.cursor()
        cursor.execute(query)

        rows = cursor.fetchall()
        column_names = [description[0] for description in cursor.description]
        return [dict(zip(column_names, row, strict=False)) for row in rows]

    def write_table(
        self,
        table_name: str,
        rows: Sequence[Mapping[str, Any]],
        if_exists: str = "append",
    ) -> int:
        """Write rows to a table and return the number of inserted rows.

        Supported ``if_exists`` values:
        - ``append``: insert rows into the table
        - ``replace``: delete existing rows before inserting
        """

        connection = self._require_connection()
        if if_exists not in {"append", "replace"}:
            raise ValueError("if_exists must be either 'append' or 'replace'")

        if not rows:
            return 0

        table = self._validate_identifier(table_name, "table_name")

        first_row_keys = list(rows[0].keys())
        if not first_row_keys:
            raise ValueError("rows must contain at least one column")

        for row in rows:
            row_keys = list(row.keys())
            if row_keys != first_row_keys:
                raise ValueError("all rows must have the same column order")

        columns = [self._validate_identifier(column, "column") for column in first_row_keys]

        placeholders = ", ".join(self._placeholder(index + 1) for index in range(len(columns)))
        columns_clause = ", ".join(columns)
        query = f"INSERT INTO {table} ({columns_clause}) VALUES ({placeholders})"

        values = [tuple(row[column] for column in first_row_keys) for row in rows]

        cursor = connection.cursor()
        if if_exists == "replace":
            cursor.execute(f"DELETE FROM {table}")

        cursor.executemany(query, values)

        if hasattr(connection, "commit"):
            connection.commit()

        return len(rows)

    def _placeholder(self, position: int) -> str:
        """Return a positional placeholder for SQL parameter binding."""

        _ = position
        return "?"

    def _require_connection(self) -> Any:
        if self._connection is None:
            raise RuntimeError("Database connection has not been established. Call connect first.")
        return self._connection

    def _validate_identifier(self, value: str, field_name: str) -> str:
        if not self._IDENTIFIER_PATTERN.match(value):
            raise ValueError(
                f"Invalid {field_name}: only letters, digits, underscores and dots are allowed"
            )
        return value
