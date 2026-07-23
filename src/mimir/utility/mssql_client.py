"""Microsoft SQL Server database utility implementation."""

from __future__ import annotations

from typing import Any

from .database_client import DatabaseClient


class MSSQLClient(DatabaseClient):
    """Utility class for connecting to and operating on MSSQL databases."""

    def connect(self, auth_method: str = "password", **kwargs: Any) -> None:
        """Connect to SQL Server.

        Supported auth methods:
        - ``password``: server, database, username, password, optional port/driver
        - ``integrated``: server, database, optional port/driver
        - ``connection_string``: connection_string
        """

        try:
            import pyodbc
        except ImportError as exc:
            raise ImportError(
                "pyodbc is required for MSSQLClient. Install dependencies first."
            ) from exc

        timeout = int(kwargs.get("timeout", 30))

        if auth_method == "connection_string":
            connection_string = kwargs.get("connection_string")
            if not connection_string:
                raise ValueError(
                    "connection_string is required for auth_method='connection_string'"
                )
            self._connection = pyodbc.connect(connection_string, timeout=timeout)
            return

        server = kwargs.get("server")
        database = kwargs.get("database")
        driver = kwargs.get("driver", "ODBC Driver 18 for SQL Server")
        port = kwargs.get("port")
        if not server or not database:
            raise ValueError("server and database are required")

        server_target = f"{server},{port}" if port else str(server)

        if auth_method == "password":
            username = kwargs.get("username")
            password = kwargs.get("password")
            if not username or not password:
                raise ValueError(
                    "username and password are required for auth_method='password'"
                )
            connection_string = (
                f"DRIVER={{{driver}}};"
                f"SERVER={server_target};"
                f"DATABASE={database};"
                f"UID={username};"
                f"PWD={password};"
                "Encrypt=yes;TrustServerCertificate=yes;"
            )
            self._connection = pyodbc.connect(connection_string, timeout=timeout)
            return

        if auth_method == "integrated":
            connection_string = (
                f"DRIVER={{{driver}}};"
                f"SERVER={server_target};"
                f"DATABASE={database};"
                "Trusted_Connection=yes;"
                "Encrypt=yes;TrustServerCertificate=yes;"
            )
            self._connection = pyodbc.connect(connection_string, timeout=timeout)
            return

        raise ValueError(f"Unsupported auth_method for MSSQL: {auth_method}")
