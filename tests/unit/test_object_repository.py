from __future__ import annotations

from unittest.mock import MagicMock

import pytest

from mimir.metadata import ObjectRecord, ObjectRepository


# ---------------------------------------------------------------------------
# ObjectRecord serialisation
# ---------------------------------------------------------------------------


@pytest.mark.unit
def test_object_record_to_dict_all_fields() -> None:
    record = ObjectRecord(
        name="customers",
        source_name="crm",
        type="table",
        description="Customer master data",
        is_active=True,
        is_current=False,
    )
    result = record.to_dict()
    assert result == {
        "name": "customers",
        "source_name": "crm",
        "type": "table",
        "description": "Customer master data",
        "is_active": True,
        "is_current": False,
    }


@pytest.mark.unit
def test_object_record_to_dict_defaults() -> None:
    record = ObjectRecord(name="orders", source_name="erp")
    result = record.to_dict()
    assert result["name"] == "orders"
    assert result["source_name"] == "erp"
    assert result["type"] is None
    assert result["description"] is None
    assert result["is_active"] is True
    assert result["is_current"] is True


@pytest.mark.unit
def test_object_record_from_dict_full_row() -> None:
    row = {
        "name": "invoices",
        "source_name": "billing",
        "type": "view",
        "description": "Invoice data",
        "is_active": 1,
        "is_current": 0,
    }
    record = ObjectRecord.from_dict(row)
    assert record.name == "invoices"
    assert record.source_name == "billing"
    assert record.type == "view"
    assert record.description == "Invoice data"
    assert record.is_active is True
    assert record.is_current is False


@pytest.mark.unit
def test_object_record_from_dict_missing_optional_fields() -> None:
    row = {
        "name": "slim",
        "source_name": "src",
        "is_active": True,
        "is_current": True,
    }
    record = ObjectRecord.from_dict(row)
    assert record.type is None
    assert record.description is None


@pytest.mark.unit
def test_object_record_roundtrip() -> None:
    original = ObjectRecord(
        name="rt",
        source_name="src",
        type="table",
        description="roundtrip",
    )
    assert ObjectRecord.from_dict(original.to_dict()) == original


# ---------------------------------------------------------------------------
# ObjectRepository helpers
# ---------------------------------------------------------------------------


def _mock_client(rows: list[dict] | None = None) -> MagicMock:
    client = MagicMock()
    client.read_table.return_value = rows or []
    return client


# ---------------------------------------------------------------------------
# insert
# ---------------------------------------------------------------------------


@pytest.mark.unit
def test_insert_calls_write_table() -> None:
    client = _mock_client(rows=[])
    repo = ObjectRepository(client)
    record = ObjectRecord(name="new_obj", source_name="src")
    repo.insert(record)
    client.write_table.assert_called_once_with("object", [record.to_dict()])


@pytest.mark.unit
def test_insert_raises_on_duplicate() -> None:
    existing_row = ObjectRecord(name="dup", source_name="src").to_dict()
    client = _mock_client(rows=[existing_row])
    repo = ObjectRepository(client)
    with pytest.raises(ValueError, match="already exists"):
        repo.insert(ObjectRecord(name="dup", source_name="src"))
    client.write_table.assert_not_called()


# ---------------------------------------------------------------------------
# update
# ---------------------------------------------------------------------------


@pytest.mark.unit
def test_update_returns_true_when_found() -> None:
    row = ObjectRecord(name="obj", source_name="src").to_dict()
    client = _mock_client(rows=[row])
    repo = ObjectRepository(client)
    result = repo.update("obj", type="view")
    assert result is True
    client.execute.assert_called_once()
    call_args = client.execute.call_args
    assert "UPDATE object SET" in call_args[0][0]
    assert "WHERE name = ?" in call_args[0][0]


@pytest.mark.unit
def test_update_returns_false_when_not_found() -> None:
    client = _mock_client(rows=[])
    repo = ObjectRepository(client)
    result = repo.update("ghost", type="view")
    assert result is False
    client.execute.assert_not_called()


@pytest.mark.unit
def test_update_returns_false_for_empty_fields() -> None:
    row = ObjectRecord(name="obj", source_name="src").to_dict()
    client = _mock_client(rows=[row])
    repo = ObjectRepository(client)
    result = repo.update("obj")
    assert result is False
    client.execute.assert_not_called()


@pytest.mark.unit
def test_update_raises_for_unknown_field() -> None:
    client = _mock_client(rows=[])
    repo = ObjectRepository(client)
    with pytest.raises(ValueError, match="non-updatable"):
        repo.update("obj", nonexistent_column="x")


@pytest.mark.unit
def test_update_raises_for_primary_key_field() -> None:
    # `name` is the positional parameter of update(); Python raises TypeError
    # before the method body can validate it as non-updatable.
    client = _mock_client(rows=[])
    repo = ObjectRepository(client)
    with pytest.raises(TypeError):
        repo.update("obj", name="new_name")


@pytest.mark.unit
def test_update_raises_for_source_name_field() -> None:
    """source_name is a FK — must NOT be updatable."""
    client = _mock_client(rows=[])
    repo = ObjectRepository(client)
    with pytest.raises(ValueError, match="non-updatable"):
        repo.update("obj", source_name="other_source")


@pytest.mark.unit
def test_update_passes_correct_params() -> None:
    row = ObjectRecord(name="obj", source_name="src").to_dict()
    client = _mock_client(rows=[row])
    repo = ObjectRepository(client)
    repo.update("obj", type="table", description="desc")
    _, kwargs = client.execute.call_args
    params = kwargs["params"]
    assert "obj" in params


# ---------------------------------------------------------------------------
# get_all
# ---------------------------------------------------------------------------


@pytest.mark.unit
def test_get_all_returns_empty_list() -> None:
    client = _mock_client(rows=[])
    repo = ObjectRepository(client)
    assert repo.get_all() == []


@pytest.mark.unit
def test_get_all_returns_records() -> None:
    rows = [
        ObjectRecord(name="a", source_name="src1").to_dict(),
        ObjectRecord(name="b", source_name="src2").to_dict(),
    ]
    client = _mock_client(rows=rows)
    repo = ObjectRepository(client)
    result = repo.get_all()
    assert len(result) == 2
    assert all(isinstance(r, ObjectRecord) for r in result)
    assert result[0].name == "a"
    assert result[1].name == "b"


# ---------------------------------------------------------------------------
# get_by_name
# ---------------------------------------------------------------------------


@pytest.mark.unit
def test_get_by_name_found() -> None:
    rows = [ObjectRecord(name="target", source_name="src", type="table").to_dict()]
    client = _mock_client(rows=rows)
    repo = ObjectRepository(client)
    record = repo.get_by_name("target")
    assert record is not None
    assert record.name == "target"
    assert record.type == "table"


@pytest.mark.unit
def test_get_by_name_not_found() -> None:
    client = _mock_client(rows=[])
    repo = ObjectRepository(client)
    assert repo.get_by_name("missing") is None


# ---------------------------------------------------------------------------
# get_by_source
# ---------------------------------------------------------------------------


@pytest.mark.unit
def test_get_by_source_returns_matching_records() -> None:
    rows = [
        ObjectRecord(name="a", source_name="crm").to_dict(),
        ObjectRecord(name="b", source_name="erp").to_dict(),
        ObjectRecord(name="c", source_name="crm").to_dict(),
    ]
    client = _mock_client(rows=rows)
    repo = ObjectRepository(client)
    result = repo.get_by_source("crm")
    assert len(result) == 2
    assert all(r.source_name == "crm" for r in result)
    assert {r.name for r in result} == {"a", "c"}


@pytest.mark.unit
def test_get_by_source_no_match_returns_empty() -> None:
    rows = [ObjectRecord(name="a", source_name="crm").to_dict()]
    client = _mock_client(rows=rows)
    repo = ObjectRepository(client)
    assert repo.get_by_source("missing") == []


# ---------------------------------------------------------------------------
# get_active / get_current
# ---------------------------------------------------------------------------


@pytest.mark.unit
def test_get_active_filters_correctly() -> None:
    rows = [
        ObjectRecord(name="active", source_name="src", is_active=True).to_dict(),
        ObjectRecord(name="inactive", source_name="src", is_active=False).to_dict(),
    ]
    client = _mock_client(rows=rows)
    repo = ObjectRepository(client)
    result = repo.get_active()
    assert len(result) == 1
    assert result[0].name == "active"


@pytest.mark.unit
def test_get_current_filters_correctly() -> None:
    rows = [
        ObjectRecord(name="current", source_name="src", is_current=True).to_dict(),
        ObjectRecord(name="old", source_name="src", is_current=False).to_dict(),
    ]
    client = _mock_client(rows=rows)
    repo = ObjectRepository(client)
    result = repo.get_current()
    assert len(result) == 1
    assert result[0].name == "current"


@pytest.mark.unit
def test_get_active_empty_table() -> None:
    client = _mock_client(rows=[])
    repo = ObjectRepository(client)
    assert repo.get_active() == []


@pytest.mark.unit
def test_get_current_empty_table() -> None:
    client = _mock_client(rows=[])
    repo = ObjectRepository(client)
    assert repo.get_current() == []
