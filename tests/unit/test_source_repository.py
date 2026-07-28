from __future__ import annotations

from unittest.mock import MagicMock

import pytest

from mimir.metadata import SourceRecord, SourceRepository


# ---------------------------------------------------------------------------
# SourceRecord serialisation
# ---------------------------------------------------------------------------


@pytest.mark.unit
def test_source_record_to_dict_all_fields() -> None:
    record = SourceRecord(
        name="crm",
        type="database",
        system="salesforce",
        owner="data-team",
        timezone="UTC",
        is_active=True,
        is_current=False,
    )
    result = record.to_dict()
    assert result == {
        "name": "crm",
        "type": "database",
        "system": "salesforce",
        "owner": "data-team",
        "timezone": "UTC",
        "is_active": True,
        "is_current": False,
    }


@pytest.mark.unit
def test_source_record_to_dict_defaults() -> None:
    record = SourceRecord(name="minimal")
    result = record.to_dict()
    assert result["name"] == "minimal"
    assert result["type"] is None
    assert result["is_active"] is True
    assert result["is_current"] is True


@pytest.mark.unit
def test_source_record_from_dict_full_row() -> None:
    row = {
        "name": "erp",
        "type": "api",
        "system": "sap",
        "owner": "finance",
        "timezone": "Europe/Amsterdam",
        "is_active": 1,
        "is_current": 0,
    }
    record = SourceRecord.from_dict(row)
    assert record.name == "erp"
    assert record.type == "api"
    assert record.system == "sap"
    assert record.owner == "finance"
    assert record.timezone == "Europe/Amsterdam"
    assert record.is_active is True
    assert record.is_current is False


@pytest.mark.unit
def test_source_record_from_dict_missing_optional_fields() -> None:
    row = {"name": "slim", "is_active": True, "is_current": True}
    record = SourceRecord.from_dict(row)
    assert record.name == "slim"
    assert record.type is None
    assert record.system is None
    assert record.owner is None
    assert record.timezone is None


@pytest.mark.unit
def test_source_record_roundtrip() -> None:
    original = SourceRecord(name="rt", type="file", system="s3", owner="ops", timezone="UTC")
    assert SourceRecord.from_dict(original.to_dict()) == original


# ---------------------------------------------------------------------------
# SourceRepository helpers
# ---------------------------------------------------------------------------


def _mock_client(rows: list[dict] | None = None) -> MagicMock:
    """Return a DatabaseClient mock whose read_table returns *rows*."""
    client = MagicMock()
    client.read_table.return_value = rows or []
    return client


# ---------------------------------------------------------------------------
# insert
# ---------------------------------------------------------------------------


@pytest.mark.unit
def test_insert_calls_write_table() -> None:
    client = _mock_client(rows=[])
    repo = SourceRepository(client)
    record = SourceRecord(name="new_source")
    repo.insert(record)
    client.write_table.assert_called_once_with("source", [record.to_dict()])


@pytest.mark.unit
def test_insert_raises_on_duplicate() -> None:
    existing_row = SourceRecord(name="dup").to_dict()
    client = _mock_client(rows=[existing_row])
    repo = SourceRepository(client)
    with pytest.raises(ValueError, match="already exists"):
        repo.insert(SourceRecord(name="dup"))
    client.write_table.assert_not_called()


# ---------------------------------------------------------------------------
# update
# ---------------------------------------------------------------------------


@pytest.mark.unit
def test_update_returns_true_when_found() -> None:
    row = SourceRecord(name="src").to_dict()
    client = _mock_client(rows=[row])
    repo = SourceRepository(client)
    result = repo.update("src", type="api")
    assert result is True
    client.execute.assert_called_once()
    call_args = client.execute.call_args
    assert "UPDATE source SET" in call_args[0][0]
    assert "WHERE name = ?" in call_args[0][0]


@pytest.mark.unit
def test_update_returns_false_when_not_found() -> None:
    client = _mock_client(rows=[])
    repo = SourceRepository(client)
    result = repo.update("ghost", type="api")
    assert result is False
    client.execute.assert_not_called()


@pytest.mark.unit
def test_update_returns_false_for_empty_fields() -> None:
    row = SourceRecord(name="src").to_dict()
    client = _mock_client(rows=[row])
    repo = SourceRepository(client)
    result = repo.update("src")
    assert result is False
    client.execute.assert_not_called()


@pytest.mark.unit
def test_update_raises_for_unknown_field() -> None:
    client = _mock_client(rows=[])
    repo = SourceRepository(client)
    with pytest.raises(ValueError, match="non-updatable"):
        repo.update("src", nonexistent_column="x")


@pytest.mark.unit
def test_update_raises_for_primary_key_field() -> None:
    # `name` is the positional parameter of update(); Python raises TypeError
    # before the method body can validate it as non-updatable.
    client = _mock_client(rows=[])
    repo = SourceRepository(client)
    with pytest.raises(TypeError):
        repo.update("src", name="new_name")


@pytest.mark.unit
def test_update_passes_correct_params() -> None:
    row = SourceRecord(name="src", type="db").to_dict()
    client = _mock_client(rows=[row])
    repo = SourceRepository(client)
    repo.update("src", type="api", owner="team")
    _, kwargs = client.execute.call_args
    params = kwargs["params"]
    assert "src" in params
    assert "api" in params or "team" in params


# ---------------------------------------------------------------------------
# get_all
# ---------------------------------------------------------------------------


@pytest.mark.unit
def test_get_all_returns_empty_list() -> None:
    client = _mock_client(rows=[])
    repo = SourceRepository(client)
    assert repo.get_all() == []


@pytest.mark.unit
def test_get_all_returns_records() -> None:
    rows = [
        SourceRecord(name="a", type="db").to_dict(),
        SourceRecord(name="b", type="api").to_dict(),
    ]
    client = _mock_client(rows=rows)
    repo = SourceRepository(client)
    result = repo.get_all()
    assert len(result) == 2
    assert all(isinstance(r, SourceRecord) for r in result)
    assert result[0].name == "a"
    assert result[1].name == "b"


# ---------------------------------------------------------------------------
# get_by_name
# ---------------------------------------------------------------------------


@pytest.mark.unit
def test_get_by_name_found() -> None:
    rows = [SourceRecord(name="target", type="file").to_dict()]
    client = _mock_client(rows=rows)
    repo = SourceRepository(client)
    record = repo.get_by_name("target")
    assert record is not None
    assert record.name == "target"
    assert record.type == "file"


@pytest.mark.unit
def test_get_by_name_not_found() -> None:
    client = _mock_client(rows=[])
    repo = SourceRepository(client)
    assert repo.get_by_name("missing") is None


# ---------------------------------------------------------------------------
# get_active / get_current
# ---------------------------------------------------------------------------


@pytest.mark.unit
def test_get_active_filters_correctly() -> None:
    rows = [
        SourceRecord(name="active", is_active=True).to_dict(),
        SourceRecord(name="inactive", is_active=False).to_dict(),
    ]
    client = _mock_client(rows=rows)
    repo = SourceRepository(client)
    result = repo.get_active()
    assert len(result) == 1
    assert result[0].name == "active"


@pytest.mark.unit
def test_get_current_filters_correctly() -> None:
    rows = [
        SourceRecord(name="current", is_current=True).to_dict(),
        SourceRecord(name="old", is_current=False).to_dict(),
    ]
    client = _mock_client(rows=rows)
    repo = SourceRepository(client)
    result = repo.get_current()
    assert len(result) == 1
    assert result[0].name == "current"


@pytest.mark.unit
def test_get_active_empty_table() -> None:
    client = _mock_client(rows=[])
    repo = SourceRepository(client)
    assert repo.get_active() == []


@pytest.mark.unit
def test_get_current_empty_table() -> None:
    client = _mock_client(rows=[])
    repo = SourceRepository(client)
    assert repo.get_current() == []
