from pathlib import Path

import pytest

from src.mimir.library import ensure_reference_database, list_sources, register_source


@pytest.mark.integration
def test_ensure_reference_database_creates_db_file(tmp_path: Path) -> None:
    db_path = tmp_path / "integration.duckdb"
    resolved = ensure_reference_database(db_path)

    assert resolved.exists()
    assert resolved.name == "integration.duckdb"


@pytest.mark.integration
def test_register_and_list_sources_round_trip(tmp_path: Path) -> None:
    db_path = tmp_path / "sources.duckdb"

    register_source("erp", "postgres", "postgres://erp", db_path=db_path)
    register_source("warehouse", "duckdb", "./warehouse.duckdb", db_path=db_path)

    sources = list_sources(db_path=db_path)

    assert len(sources) == 2
    assert sources[0]["source_name"] == "erp"
    assert sources[1]["source_name"] == "warehouse"
