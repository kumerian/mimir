from pathlib import Path

import pytest

from src.mimir.library import get_reference_db_path


@pytest.mark.unit
def test_get_reference_db_path_default_points_to_repo_data_dir() -> None:
    resolved = get_reference_db_path()
    assert resolved.name == "reference.duckdb"
    assert resolved.parent.name == "data"


@pytest.mark.unit
def test_get_reference_db_path_resolves_custom_path(tmp_path: Path) -> None:
    custom_db = tmp_path / "custom.duckdb"
    resolved = get_reference_db_path(custom_db)
    assert resolved == custom_db.resolve()
