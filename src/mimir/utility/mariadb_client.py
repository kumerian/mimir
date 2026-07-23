"""MariaDB database utility implementation."""

from __future__ import annotations

from typing import Any

from .database_client import DatabaseClient


class MariaDBClient(DatabaseClient):
    """Utility class for connecting to and operating on MariaDB databases."""

    def connect(self, auth_method: str = "password", **kwargs: Any) -> None:
        """Connect to MariaDB.

        Supported auth methods:
        - ``password``: host, database, username, password, optional port
        - ``unix_socket``: unix_socket, database, username, optional password
        """

        try:
            import mariadb
        except ImportError as exc:
            raise ImportError(
                "mariadb is required for MariaDBClient. Install dependencies first."
            ) from exc

        if auth_method == "password":
            host = kwargs.get("host")
            database = kwargs.get("database")
            username = kwargs.get("username")
            password = kwargs.get("password")
            port = int(kwargs.get("port", 3306))

            if not host or not database or not username or not password:
                raise ValueError("host, database, username and password are required")

            self._connection = mariadb.connect(
                host=host,
                user=username,
                password=password,
                database=database,
                port=port,
            )
            return

        if auth_method == "unix_socket":
            unix_socket = kwargs.get("unix_socket")
            database = kwargs.get("database")
            username = kwargs.get("username")
            password = kwargs.get("password")

            if not unix_socket or not database or not username:
                raise ValueError("unix_socket, database and username are required")

            self._connection = mariadb.connect(
                unix_socket=unix_socket,
                user=username,
                password=password,
                database=database,
            )
            return

        raise ValueError(f"Unsupported auth_method for MariaDB: {auth_method}")
