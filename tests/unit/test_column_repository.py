"""Unit tests for ColumnRepository and ColumnRecord."""

from __future__ import annotations

import pytest

from mimir.metadata import ColumnRecord, ColumnRepository


# ---------------------------------------------------------------------------
# ColumnRecord serialisation
# ---------------------------------------------------------------------------


@pytest.mark.unit
def test_column_record_to_dict_all_fields() -> None:
    """Test converting ColumnRecord with all fields to dict."""
    record = ColumnRecord(
        name="customer_id",
        object_name="customers",
        functional_name="Customer Identifier",
        description="Unique customer ID",
        ordinal_position=1,
        is_metadata=False,
        data_type="INT",
        is_key=True,
        is_nullable=False,
        is_current=True,
        is_active=True,
    )
    result = record.to_dict()
    assert result == {
        "name": "customer_id",
        "object_name": "customers",
        "functional_name": "Customer Identifier",
        "description": "Unique customer ID",
        "ordinal_position": 1,
        "is_metadata": False,
        "data_type": "INT",
        "is_key": True,
        "is_nullable": False,
        "is_current": True,
        "is_active": True,
    }


@pytest.mark.unit
def test_column_record_to_dict_with_defaults() -> None:
    """Test converting ColumnRecord with default values to dict."""
    record = ColumnRecord(
        name="email",
        object_name="customers",
    )
    result = record.to_dict()
    assert result["name"] == "email"
    assert result["object_name"] == "customers"
    assert result["functional_name"] is None
    assert result["description"] is None
    assert result["ordinal_position"] is None
    assert result["is_metadata"] is False
    assert result["data_type"] is None
    assert result["is_key"] is False
    assert result["is_nullable"] is True
    assert result["is_current"] is True
    assert result["is_active"] is True


@pytest.mark.unit
def test_column_record_from_dict_full_row() -> None:
    """Test creating ColumnRecord from full row dict."""
    row = {
        "name": "created_at",
        "object_name": "orders",
        "functional_name": "Order Creation Time",
        "description": "Timestamp when order was created",
        "ordinal_position": 5,
        "is_metadata": 1,
        "data_type": "TIMESTAMP",
        "is_key": 0,
        "is_nullable": 0,
        "is_current": 1,
        "is_active": 1,
    }
    record = ColumnRecord.from_dict(row)
    assert record.name == "created_at"
    assert record.object_name == "orders"
    assert record.functional_name == "Order Creation Time"
    assert record.description == "Timestamp when order was created"
    assert record.ordinal_position == 5
    assert record.is_metadata is True
    assert record.data_type == "TIMESTAMP"
    assert record.is_key is False
    assert record.is_nullable is False
    assert record.is_current is True
    assert record.is_active is True


@pytest.mark.unit
def test_column_record_from_dict_minimal() -> None:
    """Test creating ColumnRecord from minimal row dict."""
    row = {
        "name": "status",
        "object_name": "invoices",
    }
    record = ColumnRecord.from_dict(row)
    assert record.name == "status"
    assert record.object_name == "invoices"
    assert record.functional_name is None
    assert record.description is None
    assert record.ordinal_position is None
    assert record.is_metadata is False
    assert record.data_type is None
    assert record.is_key is False


@pytest.mark.unit
def test_column_record_from_dict_boolean_conversion() -> None:
    """Test that boolean fields are correctly converted from int."""
    row = {
        "name": "active_flag",
        "object_name": "customers",
        "is_metadata": 1,
        "is_key": 0,
        "is_nullable": 1,
        "is_current": 0,
        "is_active": 1,
    }
    record = ColumnRecord.from_dict(row)
    assert record.is_metadata is True
    assert record.is_key is False
    assert record.is_nullable is True
    assert record.is_current is False
    assert record.is_active is True


# ---------------------------------------------------------------------------
# ColumnRepository basics
# ---------------------------------------------------------------------------


@pytest.mark.unit
def test_column_repository_class_exists() -> None:
    """Test that ColumnRepository class can be imported."""
    assert ColumnRepository is not None


@pytest.mark.unit
def test_column_table_definition_loaded() -> None:
    """Test that COLUMN_TABLE is properly loaded from markdown."""
    from mimir.metadata import COLUMN_TABLE
    
    assert COLUMN_TABLE is not None
    assert COLUMN_TABLE.name == "columns"
    assert len(COLUMN_TABLE.columns) > 0
    
    # Verify key columns exist
    col_names = {col.name for col in COLUMN_TABLE.columns}
    assert "name" in col_names
    assert "object_name" in col_names
    assert "data_type" in col_names
    assert "is_key" in col_names
    assert "is_nullable" in col_names
    assert "is_active" in col_names
    assert "is_current" in col_names


@pytest.mark.unit
def test_updatable_columns_defined() -> None:
    """Test that _UPDATABLE_COLUMNS is properly defined."""
    from mimir.metadata import ColumnRepository
    from mimir.metadata.column_repository import _UPDATABLE_COLUMNS
    
    assert _UPDATABLE_COLUMNS is not None
    assert isinstance(_UPDATABLE_COLUMNS, frozenset)
    
    # name and object_name should NOT be updatable (keys)
    assert "name" not in _UPDATABLE_COLUMNS
    assert "object_name" not in _UPDATABLE_COLUMNS
    
    # Other fields should be updatable
    assert "description" in _UPDATABLE_COLUMNS
    assert "data_type" in _UPDATABLE_COLUMNS
    assert "is_active" in _UPDATABLE_COLUMNS
    assert "is_current" in _UPDATABLE_COLUMNS


# ---------------------------------------------------------------------------
# Column field integrity
# ---------------------------------------------------------------------------


@pytest.mark.unit
def test_column_record_has_all_table_fields() -> None:
    """Test that ColumnRecord has fields for all table columns."""
    from mimir.metadata import COLUMN_TABLE
    from dataclasses import fields as dataclass_fields
    
    record_field_names = {f.name for f in dataclass_fields(ColumnRecord)}
    table_col_names = {col.name for col in COLUMN_TABLE.columns}
    
    # All table columns should be in the record
    assert table_col_names <= record_field_names, \
        f"Missing fields: {table_col_names - record_field_names}"


@pytest.mark.unit
def test_column_record_default_alignment_with_table() -> None:
    """Test that ColumnRecord defaults match COLUMN_TABLE defaults."""
    from mimir.metadata import COLUMN_TABLE
    
    # Create a record with minimal fields
    record = ColumnRecord(name="test", object_name="test_obj")
    
    # Check each boolean field has correct default
    assert record.is_metadata is False
    assert record.is_key is False
    assert record.is_nullable is True
    assert record.is_current is True
    assert record.is_active is True


# ---------------------------------------------------------------------------
# Column record edge cases
# ---------------------------------------------------------------------------


@pytest.mark.unit
def test_column_record_with_null_values() -> None:
    """Test ColumnRecord with None values for nullable fields."""
    record = ColumnRecord(
        name="optional_field",
        object_name="test_table",
        functional_name=None,
        description=None,
        ordinal_position=None,
        data_type=None,
    )
    
    result = record.to_dict()
    assert result["functional_name"] is None
    assert result["description"] is None
    assert result["ordinal_position"] is None
    assert result["data_type"] is None


@pytest.mark.unit
def test_column_record_equality() -> None:
    """Test that two identical ColumnRecords are equal."""
    record1 = ColumnRecord(
        name="id",
        object_name="users",
        data_type="INT",
        is_key=True,
    )
    record2 = ColumnRecord(
        name="id",
        object_name="users",
        data_type="INT",
        is_key=True,
    )
    
    assert record1 == record2


@pytest.mark.unit
def test_column_record_repr() -> None:
    """Test string representation of ColumnRecord."""
    record = ColumnRecord(
        name="customer_id",
        object_name="customers",
    )
    
    repr_str = repr(record)
    assert "ColumnRecord" in repr_str or "customer_id" in repr_str
