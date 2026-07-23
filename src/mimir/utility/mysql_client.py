"""MySQL database utility implementation."""

from __future__ import annotations

from typing import Any

from .database_client import DatabaseClient


class MySQLClient(DatabaseClient):
    """Utility class for connecting to and operating on MySQL databases."""

    def connect(self, auth_method: str = "password", **kwargs: Any) -> None:
        """Connect to MySQL.

        Supported auth methods:
        - ``password``: host, database, username, password, optional port
        - ``ssl``: host, database, username, password, ssl dict, optional port
        """

        try:
            import pymysql
        except ImportError as exc:
            raise ImportError(
                "pymysql is required for MySQLClient. Install dependencies first."
            ) from exc

        host = kwargs.get("host")
        database = kwargs.get("database")
        username = kwargs.get("username")
        password = kwargs.get("password")
        port = int(kwargs.get("port", 3306))

        if auth_method == "password":
            if not host or not database or not username or not password:
                raise ValueError("host, database, username and password are required")

            self._connection = pymysql.connect(
                host=host,
                user=username,
                password=password,
                database=database,
                port=port,
            )
            return

        if auth_method == "ssl":
            ssl = kwargs.get("ssl")
            if not host or not database or not username or not password or not ssl:
                raise ValueError("host, database, username, password and ssl are required")

            self._connection = pymysql.connect(
                host=host,
                user=username,
                password=password,
                database=database,
                port=port,
                ssl=ssl,
            )
            return

        raise ValueError(f"Unsupported auth_method for MySQL: {auth_method}")

    def _placeholder(self, position: int) -> str:
        _ = position
        return "%s"
