"""Integration tests for ColumnRepository database operations."""

from __future__ import annotations

import pytest

from mimir.metadata import ColumnRecord, ColumnRepository, COLUMN_TABLE
from mimir.utility import DuckDBClient


@pytest.fixture()
def column_repo(tmp_path) -> ColumnRepository:
    """Create a ColumnRepository with database initialized."""
    client = DuckDBClient()
    client.connect(auth_method="path", database=str(tmp_path / "test.duckdb"))
    
    # Create the columns table
    from mimir.metadata.table_creator import MetadataTableCreator
    creator = MetadataTableCreator(client)
    creator.create_table(COLUMN_TABLE)
    
    return ColumnRepository(client)


# ---------------------------------------------------------------------------
# Column insertion tests
# ---------------------------------------------------------------------------


@pytest.mark.integration
def test_insert_single_column_record(column_repo: ColumnRepository) -> None:
    """Test inserting a single column record."""
    record = ColumnRecord(
        name="email",
        object_name="customers",
        data_type="VARCHAR",
        description="Customer email address",
    )
    
    column_repo.insert_record(record)
    
    # Verify the record exists
    columns = column_repo.get_all_columns()
    assert len(columns) == 1
    assert columns[0].name == "email"
    assert columns[0].object_name == "customers"


@pytest.mark.integration
def test_insert_multiple_column_records(column_repo: ColumnRepository) -> None:
    """Test inserting multiple column records."""
    records = [
        ColumnRecord(name="id", object_name="users", is_key=True, is_nullable=False),
        ColumnRecord(name="username", object_name="users", data_type="VARCHAR"),
        ColumnRecord(name="email", object_name="users", data_type="VARCHAR"),
        ColumnRecord(name="created_at", object_name="users", data_type="TIMESTAMP"),
    ]
    
    for record in records:
        column_repo.insert_record(record)
    
    columns = column_repo.get_all_columns()
    assert len(columns) == 4


@pytest.mark.integration
def test_insert_column_with_all_fields(column_repo: ColumnRepository) -> None:
    """Test inserting a column with all fields populated."""
    record = ColumnRecord(
        name="order_id",
        object_name="orders",
        functional_name="Order Identifier",
        description="Unique order ID",
        ordinal_position=1,
        is_metadata=False,
        data_type="BIGINT",
        is_key=True,
        is_nullable=False,
        is_current=True,
        is_active=True,
    )
    
    column_repo.insert_record(record)
    
    columns = column_repo.get_all_columns()
    assert len(columns) == 1
    col = columns[0]
    assert col.name == "order_id"
    assert col.functional_name == "Order Identifier"
    assert col.ordinal_position == 1
    assert col.is_key is True
    assert col.is_nullable is False


# ---------------------------------------------------------------------------
# Column retrieval tests
# ---------------------------------------------------------------------------


@pytest.mark.integration
def test_get_all_columns_empty(column_repo: ColumnRepository) -> None:
    """Test getting all columns from empty table."""
    columns = column_repo.get_all_columns()
    assert len(columns) == 0


@pytest.mark.integration
def test_get_columns_for_object(column_repo: ColumnRepository) -> None:
    """Test retrieving columns for a specific object."""
    # Insert columns for multiple objects
    column_repo.insert_record(ColumnRecord(name="id", object_name="customers"))
    column_repo.insert_record(ColumnRecord(name="name", object_name="customers"))
    column_repo.insert_record(ColumnRecord(name="email", object_name="customers"))
    
    column_repo.insert_record(ColumnRecord(name="id", object_name="orders"))
    column_repo.insert_record(ColumnRecord(name="customer_id", object_name="orders"))
    
    # Get columns for customers object
    customer_cols = column_repo.get_columns_for_object("customers")
    assert len(customer_cols) == 3
    names = {c.name for c in customer_cols}
    assert names == {"id", "name", "email"}
    
    # Get columns for orders object
    order_cols = column_repo.get_columns_for_object("orders")
    assert len(order_cols) == 2
    names = {c.name for c in order_cols}
    assert names == {"id", "customer_id"}


@pytest.mark.integration
def test_get_columns_for_nonexistent_object(column_repo: ColumnRepository) -> None:
    """Test retrieving columns for a nonexistent object returns empty list."""
    result = column_repo.get_columns_for_object("nonexistent")
    assert result == []


# ---------------------------------------------------------------------------
# Column update tests
# ---------------------------------------------------------------------------


@pytest.mark.integration
def test_update_column_description(column_repo: ColumnRepository) -> None:
    """Test updating a column's description."""
    record = ColumnRecord(name="status", object_name="orders", description="Old description")
    column_repo.insert_record(record)
    
    column_repo.update_record("status", "orders", description="New description")
    
    columns = column_repo.get_columns_for_object("orders")
    assert len(columns) == 1
    assert columns[0].description == "New description"


@pytest.mark.integration
def test_update_column_active_flag(column_repo: ColumnRepository) -> None:
    """Test updating a column's active flag."""
    record = ColumnRecord(name="legacy_field", object_name="customers", is_active=True)
    column_repo.insert_record(record)
    
    column_repo.update_record("legacy_field", "customers", is_active=False)
    
    columns = column_repo.get_columns_for_object("customers")
    assert columns[0].is_active is False


@pytest.mark.integration
def test_update_column_data_type(column_repo: ColumnRepository) -> None:
    """Test updating a column's data type."""
    record = ColumnRecord(name="age", object_name="customers", data_type="INT")
    column_repo.insert_record(record)
    
    column_repo.update_record("age", "customers", data_type="SMALLINT")
    
    columns = column_repo.get_columns_for_object("customers")
    assert columns[0].data_type == "SMALLINT"


@pytest.mark.integration
def test_update_multiple_fields(column_repo: ColumnRepository) -> None:
    """Test updating multiple fields at once."""
    record = ColumnRecord(
        name="created_date",
        object_name="orders",
        description="Old description",
        is_active=True,
    )
    column_repo.insert_record(record)
    
    column_repo.update_record(
        "created_date",
        "orders",
        description="New description",
        is_active=False,
        is_current=False,
    )
    
    columns = column_repo.get_columns_for_object("orders")
    assert columns[0].description == "New description"
    assert columns[0].is_active is False
    assert columns[0].is_current is False


# ---------------------------------------------------------------------------
# Column deletion tests
# ---------------------------------------------------------------------------


@pytest.mark.integration
def test_delete_column_record(column_repo: ColumnRepository) -> None:
    """Test soft-deleting a column record (marking as inactive/not current)."""
    record = ColumnRecord(name="to_delete", object_name="test_table")
    column_repo.insert_record(record)
    
    columns = column_repo.get_all_columns()
    assert len(columns) == 1
    assert columns[0].is_active is True
    assert columns[0].is_current is True
    
    # Delete marks the record as inactive and not current (soft delete)
    column_repo.delete_record("to_delete", "test_table")
    
    # Record still exists in DB but is marked as deleted
    columns = column_repo.get_all_columns()
    assert len(columns) == 1
    assert columns[0].is_active is False
    assert columns[0].is_current is False


@pytest.mark.integration
def test_delete_column_preserves_others(column_repo: ColumnRepository) -> None:
    """Test that deleting one column doesn't affect others."""
    column_repo.insert_record(ColumnRecord(name="col1", object_name="table1"))
    column_repo.insert_record(ColumnRecord(name="col2", object_name="table1"))
    column_repo.insert_record(ColumnRecord(name="col3", object_name="table2"))
    
    column_repo.delete_record("col1", "table1")
    
    # All records still exist, but col1 is marked as deleted
    remaining = column_repo.get_all_columns()
    assert len(remaining) == 3
    
    # Check that col1 is marked as deleted
    col1 = [c for c in remaining if c.name == "col1"][0]
    assert col1.is_active is False
    assert col1.is_current is False
    
    # Check that other columns are still active
    col2 = [c for c in remaining if c.name == "col2"][0]
    col3 = [c for c in remaining if c.name == "col3"][0]
    assert col2.is_active is True
    assert col3.is_active is True


# ---------------------------------------------------------------------------
# Column persistence tests
# ---------------------------------------------------------------------------


@pytest.mark.integration
def test_columns_persist_after_read(column_repo: ColumnRepository) -> None:
    """Test that columns persisted in DB can be read back."""
    original = ColumnRecord(
        name="persistent",
        object_name="test_table",
        data_type="VARCHAR",
        description="Persistent column",
        ordinal_position=1,
    )
    
    column_repo.insert_record(original)
    
    # Get from DB
    columns = column_repo.get_columns_for_object("test_table")
    assert len(columns) == 1
    
    restored = columns[0]
    assert restored.name == original.name
    assert restored.object_name == original.object_name
    assert restored.data_type == original.data_type
    assert restored.description == original.description
    assert restored.ordinal_position == original.ordinal_position


@pytest.mark.integration
def test_all_column_fields_persist(column_repo: ColumnRepository) -> None:
    """Test that all column fields persist correctly."""
    original = ColumnRecord(
        name="full_column",
        object_name="test_table",
        functional_name="Full Test Column",
        description="A column with all fields",
        ordinal_position=5,
        is_metadata=True,
        data_type="DECIMAL",
        is_key=False,
        is_nullable=False,
        is_current=False,
        is_active=True,
    )
    
    column_repo.insert_record(original)
    restored = column_repo.get_columns_for_object("test_table")[0]
    
    # Verify all fields
    assert restored.name == "full_column"
    assert restored.object_name == "test_table"
    assert restored.functional_name == "Full Test Column"
    assert restored.description == "A column with all fields"
    assert restored.ordinal_position == 5
    assert restored.is_metadata is True
    assert restored.data_type == "DECIMAL"
    assert restored.is_key is False
    assert restored.is_nullable is False
    assert restored.is_current is False
    assert restored.is_active is True


# ---------------------------------------------------------------------------
# Column edge cases
# ---------------------------------------------------------------------------


@pytest.mark.integration
def test_columns_with_special_characters(column_repo: ColumnRepository) -> None:
    """Test columns with special characters in names."""
    record = ColumnRecord(
        name="col_with_underscore",
        object_name="table_name",
        description="Column with 'quotes' and \"double quotes\"",
    )
    
    column_repo.insert_record(record)
    
    retrieved = column_repo.get_columns_for_object("table_name")[0]
    assert retrieved.name == "col_with_underscore"
    assert "quotes" in retrieved.description


@pytest.mark.integration
def test_many_columns_in_single_object(column_repo: ColumnRepository) -> None:
    """Test handling many columns for a single object."""
    object_name = "large_table"
    
    # Insert 50 columns
    for i in range(50):
        record = ColumnRecord(
            name=f"col_{i:02d}",
            object_name=object_name,
            ordinal_position=i,
        )
        column_repo.insert_record(record)
    
    columns = column_repo.get_columns_for_object(object_name)
    assert len(columns) == 50
    
    # Verify they can be sorted by position
    sorted_cols = sorted(columns, key=lambda c: c.ordinal_position or 0)
    for i, col in enumerate(sorted_cols):
        assert col.name == f"col_{i:02d}"
