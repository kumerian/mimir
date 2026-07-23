"""PostgreSQL database utility implementation."""

from __future__ import annotations

from typing import Any

from .database_client import DatabaseClient


class PostgreSQLClient(DatabaseClient):
    """Utility class for connecting to and operating on PostgreSQL databases."""

    def connect(self, auth_method: str = "password", **kwargs: Any) -> None:
        """Connect to PostgreSQL.

        Supported auth methods:
        - ``password``: host, database, username, password, optional port/sslmode
        - ``dsn``: dsn
        """

        try:
            import psycopg
        except ImportError as exc:
            raise ImportError(
                "psycopg is required for PostgreSQLClient. Install dependencies first."
            ) from exc

        if auth_method == "dsn":
            dsn = kwargs.get("dsn")
            if not dsn:
                raise ValueError("dsn is required for auth_method='dsn'")
            self._connection = psycopg.connect(dsn)
            return

        if auth_method == "password":
            host = kwargs.get("host")
            database = kwargs.get("database")
            username = kwargs.get("username")
            password = kwargs.get("password")
            port = int(kwargs.get("port", 5432))
            sslmode = kwargs.get("sslmode")

            if not host or not database or not username or not password:
                raise ValueError("host, database, username and password are required")

            connect_kwargs: dict[str, Any] = {
                "host": host,
                "dbname": database,
                "user": username,
                "password": password,
                "port": port,
            }
            if sslmode:
                connect_kwargs["sslmode"] = sslmode

            self._connection = psycopg.connect(**connect_kwargs)
            return

        raise ValueError(f"Unsupported auth_method for PostgreSQL: {auth_method}")

    def _placeholder(self, position: int) -> str:
        _ = position
        return "%s"
