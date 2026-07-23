"""Oracle database utility implementation."""

from __future__ import annotations

from typing import Any

from .database_client import DatabaseClient


class OracleClient(DatabaseClient):
    """Utility class for connecting to and operating on Oracle databases."""

    def connect(self, auth_method: str = "password", **kwargs: Any) -> None:
        """Connect to Oracle.

        Supported auth methods:
        - ``password``: username, password, dsn
        - ``wallet``: username, password, dsn, config_dir, wallet_location, optional wallet_password
        - ``external``: dsn (uses external authentication)
        """

        try:
            import oracledb
        except ImportError as exc:
            raise ImportError(
                "oracledb is required for OracleClient. Install dependencies first."
            ) from exc

        if auth_method == "password":
            username = kwargs.get("username")
            password = kwargs.get("password")
            dsn = kwargs.get("dsn")
            if not username or not password or not dsn:
                raise ValueError("username, password and dsn are required")

            self._connection = oracledb.connect(user=username, password=password, dsn=dsn)
            return

        if auth_method == "wallet":
            username = kwargs.get("username")
            password = kwargs.get("password")
            dsn = kwargs.get("dsn")
            config_dir = kwargs.get("config_dir")
            wallet_location = kwargs.get("wallet_location")
            wallet_password = kwargs.get("wallet_password")
            if not username or not password or not dsn or not config_dir or not wallet_location:
                raise ValueError(
                    "username, password, dsn, config_dir and wallet_location are required"
                )

            self._connection = oracledb.connect(
                user=username,
                password=password,
                dsn=dsn,
                config_dir=config_dir,
                wallet_location=wallet_location,
                wallet_password=wallet_password,
            )
            return

        if auth_method == "external":
            dsn = kwargs.get("dsn")
            if not dsn:
                raise ValueError("dsn is required for auth_method='external'")

            self._connection = oracledb.connect(dsn=dsn, externalauth=True)
            return

        raise ValueError(f"Unsupported auth_method for Oracle: {auth_method}")

    def _placeholder(self, position: int) -> str:
        return f":{position}"
