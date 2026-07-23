"""DuckDB database utility implementation."""

from __future__ import annotations

from typing import Any

from .database_client import DatabaseClient


class DuckDBClient(DatabaseClient):
    """Utility class for connecting to and operating on DuckDB databases."""

    def connect(self, auth_method: str = "path", **kwargs: Any) -> None:
        """Connect to DuckDB.

        Supported auth methods:
        - ``path``: use ``database`` (default ``:memory:``) and optional ``read_only``
        - ``config``: use ``database`` and optional ``config`` dict
        """

        try:
            import duckdb
        except ImportError as exc:
            raise ImportError(
                "duckdb is required for DuckDBClient. Install dependencies first."
            ) from exc

        if auth_method == "path":
            database = kwargs.get("database", ":memory:")
            read_only = bool(kwargs.get("read_only", False))
            self._connection = duckdb.connect(database=database, read_only=read_only)
            return

        if auth_method == "config":
            database = kwargs.get("database", ":memory:")
            config = kwargs.get("config", {})
            self._connection = duckdb.connect(database=database, config=config)
            return

        raise ValueError(f"Unsupported auth_method for DuckDB: {auth_method}")
