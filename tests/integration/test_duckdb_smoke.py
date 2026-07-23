import duckdb
import pytest


@pytest.mark.integration
def test_duckdb_temp_database_round_trip(tmp_path) -> None:
    db_path = tmp_path / "integration.duckdb"

    with duckdb.connect(str(db_path)) as connection:
        result = connection.execute("SELECT 1").fetchone()

    assert result == (1,)
