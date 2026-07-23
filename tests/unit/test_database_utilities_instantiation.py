import pytest

from mimir.utility import (
    DuckDBClient,
    MariaDBClient,
    MSSQLClient,
    MySQLClient,
    OracleClient,
    PostgreSQLClient,
)


@pytest.mark.unit
def test_all_database_utilities_can_be_instantiated_empty() -> None:
    utilities = [
        DuckDBClient(),
        MSSQLClient(),
        PostgreSQLClient(),
        MariaDBClient(),
        MySQLClient(),
        OracleClient(),
    ]

    assert all(utility.is_connected is False for utility in utilities)
