from __future__ import annotations

import pytest

from mimir.metadata import ObjectRecord, ObjectRepository, SourceRecord, SourceRepository
from mimir.utility import DuckDBClient


@pytest.fixture()
def repo(tmp_path):
    client = DuckDBClient()
    client.connect(auth_method="path", database=str(tmp_path / "test.duckdb"))
    # Create source table first (FK dependency)
    source_repo = SourceRepository(client)
    source_repo.create_table()
    source_repo.insert(SourceRecord(name="crm"))
    source_repo.insert(SourceRecord(name="erp"))
    r = ObjectRepository(client)
    r.create_table()
    return r


# ---------------------------------------------------------------------------
# Round-trip: insert → read back
# ---------------------------------------------------------------------------


@pytest.mark.integration
def test_insert_and_get_all(repo: ObjectRepository) -> None:
    record = ObjectRecord(name="customers", source_name="crm", type="table")
    repo.insert(record)
    results = repo.get_all()
    assert len(results) == 1
    assert results[0].name == "customers"
    assert results[0].source_name == "crm"
    assert results[0].type == "table"


@pytest.mark.integration
def test_get_by_name_found(repo: ObjectRepository) -> None:
    repo.insert(ObjectRecord(name="invoices", source_name="erp", type="view"))
    result = repo.get_by_name("invoices")
    assert result is not None
    assert result.name == "invoices"
    assert result.type == "view"


@pytest.mark.integration
def test_get_by_name_not_found(repo: ObjectRepository) -> None:
    assert repo.get_by_name("nonexistent") is None


# ---------------------------------------------------------------------------
# Duplicate insert
# ---------------------------------------------------------------------------


@pytest.mark.integration
def test_insert_duplicate_raises_value_error(repo: ObjectRepository) -> None:
    repo.insert(ObjectRecord(name="dup", source_name="crm"))
    with pytest.raises(ValueError, match="already exists"):
        repo.insert(ObjectRecord(name="dup", source_name="crm"))


# ---------------------------------------------------------------------------
# Update → verify persisted
# ---------------------------------------------------------------------------


@pytest.mark.integration
def test_update_persists_changes(repo: ObjectRepository) -> None:
    repo.insert(ObjectRecord(name="obj", source_name="crm", type="table"))
    updated = repo.update("obj", type="view", description="Updated desc")
    assert updated is True
    record = repo.get_by_name("obj")
    assert record is not None
    assert record.type == "view"
    assert record.description == "Updated desc"


@pytest.mark.integration
def test_update_not_found_returns_false(repo: ObjectRepository) -> None:
    result = repo.update("ghost", type="view")
    assert result is False


@pytest.mark.integration
def test_update_empty_fields_returns_false(repo: ObjectRepository) -> None:
    repo.insert(ObjectRecord(name="obj", source_name="crm"))
    result = repo.update("obj")
    assert result is False


# ---------------------------------------------------------------------------
# get_by_source
# ---------------------------------------------------------------------------


@pytest.mark.integration
def test_get_by_source_returns_correct_records(repo: ObjectRepository) -> None:
    repo.insert(ObjectRecord(name="customers", source_name="crm"))
    repo.insert(ObjectRecord(name="accounts", source_name="crm"))
    repo.insert(ObjectRecord(name="invoices", source_name="erp"))
    results = repo.get_by_source("crm")
    assert len(results) == 2
    assert all(r.source_name == "crm" for r in results)
    assert {r.name for r in results} == {"customers", "accounts"}


@pytest.mark.integration
def test_get_by_source_no_match_returns_empty(repo: ObjectRepository) -> None:
    repo.insert(ObjectRecord(name="obj", source_name="crm"))
    assert repo.get_by_source("missing") == []


# ---------------------------------------------------------------------------
# get_active / get_current
# ---------------------------------------------------------------------------


@pytest.mark.integration
def test_get_active_returns_only_active(repo: ObjectRepository) -> None:
    repo.insert(ObjectRecord(name="active_obj", source_name="crm", is_active=True))
    repo.insert(ObjectRecord(name="inactive_obj", source_name="crm", is_active=False))
    active = repo.get_active()
    names = [r.name for r in active]
    assert "active_obj" in names
    assert "inactive_obj" not in names


@pytest.mark.integration
def test_get_current_returns_only_current(repo: ObjectRepository) -> None:
    repo.insert(ObjectRecord(name="current_obj", source_name="crm", is_current=True))
    repo.insert(ObjectRecord(name="old_obj", source_name="crm", is_current=False))
    current = repo.get_current()
    names = [r.name for r in current]
    assert "current_obj" in names
    assert "old_obj" not in names


@pytest.mark.integration
def test_get_all_multiple_records(repo: ObjectRepository) -> None:
    repo.insert(ObjectRecord(name="a", source_name="crm"))
    repo.insert(ObjectRecord(name="b", source_name="erp"))
    repo.insert(ObjectRecord(name="c", source_name="crm"))
    results = repo.get_all()
    assert len(results) == 3
    assert {r.name for r in results} == {"a", "b", "c"}
