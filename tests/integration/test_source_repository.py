from __future__ import annotations

import pytest

from mimir.metadata import SourceRecord, SourceRepository
from mimir.utility import DuckDBClient


@pytest.fixture()
def repo(tmp_path):
    client = DuckDBClient()
    client.connect(auth_method="path", database=str(tmp_path / "test.duckdb"))
    r = SourceRepository(client)
    r.create_table()
    return r


# ---------------------------------------------------------------------------
# Round-trip: insert → read back
# ---------------------------------------------------------------------------


@pytest.mark.integration
def test_insert_and_get_all(repo: SourceRepository) -> None:
    record = SourceRecord(name="crm", type="database", owner="data-team")
    repo.insert(record)
    results = repo.get_all()
    assert len(results) == 1
    assert results[0].name == "crm"
    assert results[0].type == "database"
    assert results[0].owner == "data-team"


@pytest.mark.integration
def test_get_by_name_found(repo: SourceRepository) -> None:
    repo.insert(SourceRecord(name="erp", type="api"))
    result = repo.get_by_name("erp")
    assert result is not None
    assert result.name == "erp"
    assert result.type == "api"


@pytest.mark.integration
def test_get_by_name_not_found(repo: SourceRepository) -> None:
    assert repo.get_by_name("nonexistent") is None


# ---------------------------------------------------------------------------
# Duplicate insert
# ---------------------------------------------------------------------------


@pytest.mark.integration
def test_insert_duplicate_raises_value_error(repo: SourceRepository) -> None:
    repo.insert(SourceRecord(name="dup"))
    with pytest.raises(ValueError, match="already exists"):
        repo.insert(SourceRecord(name="dup"))


# ---------------------------------------------------------------------------
# Update → verify persisted
# ---------------------------------------------------------------------------


@pytest.mark.integration
def test_update_persists_changes(repo: SourceRepository) -> None:
    repo.insert(SourceRecord(name="src", type="file"))
    updated = repo.update("src", type="api", owner="ops-team")
    assert updated is True
    record = repo.get_by_name("src")
    assert record is not None
    assert record.type == "api"
    assert record.owner == "ops-team"


@pytest.mark.integration
def test_update_not_found_returns_false(repo: SourceRepository) -> None:
    result = repo.update("ghost", type="api")
    assert result is False


@pytest.mark.integration
def test_update_empty_fields_returns_false(repo: SourceRepository) -> None:
    repo.insert(SourceRecord(name="src"))
    result = repo.update("src")
    assert result is False


# ---------------------------------------------------------------------------
# get_active / get_current
# ---------------------------------------------------------------------------


@pytest.mark.integration
def test_get_active_returns_only_active(repo: SourceRepository) -> None:
    repo.insert(SourceRecord(name="active_src", is_active=True))
    repo.insert(SourceRecord(name="inactive_src", is_active=False))
    active = repo.get_active()
    names = [r.name for r in active]
    assert "active_src" in names
    assert "inactive_src" not in names


@pytest.mark.integration
def test_get_current_returns_only_current(repo: SourceRepository) -> None:
    repo.insert(SourceRecord(name="current_src", is_current=True))
    repo.insert(SourceRecord(name="old_src", is_current=False))
    current = repo.get_current()
    names = [r.name for r in current]
    assert "current_src" in names
    assert "old_src" not in names


@pytest.mark.integration
def test_get_all_multiple_records(repo: SourceRepository) -> None:
    repo.insert(SourceRecord(name="a"))
    repo.insert(SourceRecord(name="b"))
    repo.insert(SourceRecord(name="c"))
    results = repo.get_all()
    assert len(results) == 3
    assert {r.name for r in results} == {"a", "b", "c"}
