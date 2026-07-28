"""Integration tests for the hierarchical Metadata tree structure."""

from __future__ import annotations

import pytest

from mimir.metadata import Metadata, Object, Source
from mimir.utility import DuckDBClient


@pytest.fixture()
def metadata_tree(tmp_path):
    """Create a Metadata tree with tables initialized."""
    client = DuckDBClient()
    client.connect(auth_method="path", database=str(tmp_path / "test.duckdb"))
    metadata = Metadata(client)
    metadata.create_tables()
    return metadata


# ---------------------------------------------------------------------------
# Metadata level tests
# ---------------------------------------------------------------------------


@pytest.mark.integration
def test_metadata_initialization(metadata_tree: Metadata) -> None:
    """Test that Metadata is properly initialized."""
    assert metadata_tree.sources == {}
    assert len(metadata_tree.get_sources()) == 0


@pytest.mark.integration
def test_add_source_creates_source(metadata_tree: Metadata) -> None:
    """Test adding a source to metadata."""
    source = metadata_tree.add_source(name="crm", type="database", owner="data-team")
    assert source.name == "crm"
    assert source.type == "database"
    assert source.owner == "data-team"
    assert source.metadata is metadata_tree


@pytest.mark.integration
def test_add_duplicate_source_raises_error(metadata_tree: Metadata) -> None:
    """Test that duplicate sources raise ValueError."""
    metadata_tree.add_source(name="crm")
    with pytest.raises(ValueError, match="already exists"):
        metadata_tree.add_source(name="crm")


@pytest.mark.integration
def test_get_source_by_name(metadata_tree: Metadata) -> None:
    """Test retrieving a source by name."""
    added_source = metadata_tree.add_source(name="erp", type="api")
    retrieved = metadata_tree.get_source("erp")
    assert retrieved is added_source
    assert retrieved.name == "erp"


@pytest.mark.integration
def test_get_source_not_found(metadata_tree: Metadata) -> None:
    """Test that getting a nonexistent source returns None."""
    assert metadata_tree.get_source("nonexistent") is None


@pytest.mark.integration
def test_get_all_sources(metadata_tree: Metadata) -> None:
    """Test retrieving all sources."""
    metadata_tree.add_source(name="crm")
    metadata_tree.add_source(name="erp")
    metadata_tree.add_source(name="warehouse")
    sources = metadata_tree.get_sources()
    assert len(sources) == 3
    names = {s.name for s in sources}
    assert names == {"crm", "erp", "warehouse"}


# ---------------------------------------------------------------------------
# Source level tests
# ---------------------------------------------------------------------------


@pytest.mark.integration
def test_add_object_to_source(metadata_tree: Metadata) -> None:
    """Test adding an object to a source."""
    source = metadata_tree.add_source(name="crm")
    obj = source.add_object(name="customers", type="table")
    assert obj.name == "customers"
    assert obj.type == "table"
    assert obj.source is source


@pytest.mark.integration
def test_add_duplicate_object_raises_error(metadata_tree: Metadata) -> None:
    """Test that duplicate objects in a source raise ValueError."""
    source = metadata_tree.add_source(name="crm")
    source.add_object(name="customers")
    with pytest.raises(ValueError, match="already exists"):
        source.add_object(name="customers")


@pytest.mark.integration
def test_get_object_by_name(metadata_tree: Metadata) -> None:
    """Test retrieving an object by name from a source."""
    source = metadata_tree.add_source(name="crm")
    added_obj = source.add_object(name="invoices", type="view")
    retrieved = source.get_object("invoices")
    assert retrieved is added_obj
    assert retrieved.name == "invoices"


@pytest.mark.integration
def test_get_object_not_found(metadata_tree: Metadata) -> None:
    """Test that getting a nonexistent object returns None."""
    source = metadata_tree.add_source(name="crm")
    assert source.get_object("nonexistent") is None


@pytest.mark.integration
def test_get_all_objects_in_source(metadata_tree: Metadata) -> None:
    """Test retrieving all objects from a source."""
    source = metadata_tree.add_source(name="crm")
    source.add_object(name="customers")
    source.add_object(name="orders")
    source.add_object(name="payments")
    objects = source.get_objects()
    assert len(objects) == 3
    names = {o.name for o in objects}
    assert names == {"customers", "orders", "payments"}


# ---------------------------------------------------------------------------
# Object level tests
# ---------------------------------------------------------------------------


@pytest.mark.integration
def test_object_has_source_reference(metadata_tree: Metadata) -> None:
    """Test that objects maintain reference to their source."""
    source = metadata_tree.add_source(name="crm")
    obj = source.add_object(name="customers")
    assert obj.source is source
    assert obj.source.name == "crm"


@pytest.mark.integration
def test_object_has_metadata_reference(metadata_tree: Metadata) -> None:
    """Test that objects can access metadata through source."""
    source = metadata_tree.add_source(name="crm")
    obj = source.add_object(name="customers")
    assert obj.source.metadata is metadata_tree


# ---------------------------------------------------------------------------
# Tree navigation tests
# ---------------------------------------------------------------------------


@pytest.mark.integration
def test_navigate_tree_upward(metadata_tree: Metadata) -> None:
    """Test navigating from object to source to metadata."""
    source = metadata_tree.add_source(name="crm", type="database")
    obj = source.add_object(name="customers", type="table")

    # Navigate upward
    assert obj.source.name == "crm"
    assert obj.source.metadata is metadata_tree
    assert obj.source.metadata.get_source("crm") is source


@pytest.mark.integration
def test_navigate_tree_downward(metadata_tree: Metadata) -> None:
    """Test navigating from metadata to source to object."""
    source = metadata_tree.add_source(name="crm")
    obj = source.add_object(name="customers")

    # Navigate downward
    retrieved_source = metadata_tree.get_source("crm")
    assert retrieved_source is source
    retrieved_obj = retrieved_source.get_object("customers")
    assert retrieved_obj is obj


# ---------------------------------------------------------------------------
# Multi-source tests
# ---------------------------------------------------------------------------


@pytest.mark.integration
def test_multiple_sources_with_objects(metadata_tree: Metadata) -> None:
    """Test multiple sources each with their own objects."""
    crm = metadata_tree.add_source(name="crm")
    erp = metadata_tree.add_source(name="erp")

    crm.add_object(name="customers")
    crm.add_object(name="orders")
    erp.add_object(name="invoices")
    erp.add_object(name="payments")

    assert len(crm.get_objects()) == 2
    assert len(erp.get_objects()) == 2
    assert len(metadata_tree.get_all_objects()) == 4


@pytest.mark.integration
def test_get_all_objects_across_sources(metadata_tree: Metadata) -> None:
    """Test retrieving all objects across all sources."""
    crm = metadata_tree.add_source(name="crm")
    erp = metadata_tree.add_source(name="erp")
    warehouse = metadata_tree.add_source(name="warehouse")

    crm.add_object(name="customers")
    crm.add_object(name="orders")
    erp.add_object(name="invoices")
    warehouse.add_object(name="fact_sales")

    all_objs = metadata_tree.get_all_objects()
    assert len(all_objs) == 4
    obj_names = {o.name for o in all_objs}
    assert obj_names == {"customers", "orders", "invoices", "fact_sales"}


# ---------------------------------------------------------------------------
# Active/Current filtering tests
# ---------------------------------------------------------------------------


@pytest.mark.integration
def test_get_active_sources(metadata_tree: Metadata) -> None:
    """Test filtering active sources."""
    metadata_tree.add_source(name="active1", is_active=True)
    metadata_tree.add_source(name="inactive", is_active=False)
    metadata_tree.add_source(name="active2", is_active=True)

    active = metadata_tree.get_active_sources()
    names = {s.name for s in active}
    assert names == {"active1", "active2"}


@pytest.mark.integration
def test_get_current_sources(metadata_tree: Metadata) -> None:
    """Test filtering current sources."""
    metadata_tree.add_source(name="current1", is_current=True)
    metadata_tree.add_source(name="old", is_current=False)
    metadata_tree.add_source(name="current2", is_current=True)

    current = metadata_tree.get_current_sources()
    names = {s.name for s in current}
    assert names == {"current1", "current2"}


@pytest.mark.integration
def test_get_active_objects_in_source(metadata_tree: Metadata) -> None:
    """Test filtering active objects in a source."""
    source = metadata_tree.add_source(name="crm")
    source.add_object(name="active_obj", is_active=True)
    source.add_object(name="inactive_obj", is_active=False)
    source.add_object(name="another_active", is_active=True)

    active = source.get_active_objects()
    names = {o.name for o in active}
    assert names == {"active_obj", "another_active"}


@pytest.mark.integration
def test_get_current_objects_in_source(metadata_tree: Metadata) -> None:
    """Test filtering current objects in a source."""
    source = metadata_tree.add_source(name="crm")
    source.add_object(name="current_obj", is_current=True)
    source.add_object(name="old_obj", is_current=False)

    current = source.get_current_objects()
    names = {o.name for o in current}
    assert names == {"current_obj"}


# ---------------------------------------------------------------------------
# Update tests
# ---------------------------------------------------------------------------


@pytest.mark.integration
def test_update_source_metadata(metadata_tree: Metadata) -> None:
    """Test updating source metadata."""
    source = metadata_tree.add_source(name="crm", type="database")
    assert source.type == "database"

    result = source.update(type="api", owner="data-team")
    assert result is True
    assert source.type == "api"
    assert source.owner == "data-team"

    # Verify persisted
    retrieved = metadata_tree.get_source("crm")
    assert retrieved.type == "api"
    assert retrieved.owner == "data-team"


@pytest.mark.integration
def test_update_object_metadata(metadata_tree: Metadata) -> None:
    """Test updating object metadata."""
    source = metadata_tree.add_source(name="crm")
    obj = source.add_object(name="customers", type="table")

    result = obj.update(type="view", description="Customer master")
    assert result is True
    assert obj.type == "view"
    assert obj.description == "Customer master"

    # Verify persisted
    retrieved = source.get_object("customers")
    assert retrieved.type == "view"
    assert retrieved.description == "Customer master"


@pytest.mark.integration
def test_load_metadata_from_database(tmp_path) -> None:
    """Test loading the entire tree from a populated database."""
    # Create and populate
    client1 = DuckDBClient()
    db_path = str(tmp_path / "test.duckdb")
    client1.connect(auth_method="path", database=db_path)

    metadata1 = Metadata(client1)
    metadata1.create_tables()
    crm = metadata1.add_source(name="crm", type="database")
    crm.add_object(name="customers")
    crm.add_object(name="orders")
    erp = metadata1.add_source(name="erp", type="api")
    erp.add_object(name="invoices")

    # Load from database with new client
    client2 = DuckDBClient()
    client2.connect(auth_method="path", database=db_path)
    metadata2 = Metadata(client2)
    metadata2.load_from_database()

    # Verify structure is reconstructed
    assert len(metadata2.get_sources()) == 2
    crm_loaded = metadata2.get_source("crm")
    assert crm_loaded is not None
    assert len(crm_loaded.get_objects()) == 2
    erp_loaded = metadata2.get_source("erp")
    assert erp_loaded is not None
    assert len(erp_loaded.get_objects()) == 1
