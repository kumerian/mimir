"""Example usage of the hierarchical Metadata tree structure.

This example demonstrates the new object-oriented, tree-based API for
managing metadata in Mimir. The Metadata, Source, and Object classes form
a natural hierarchy that simplifies metadata operations.
"""

from mimir.metadata import Metadata
from mimir.utility import DuckDBClient


def example_basic_tree_structure() -> None:
    """Basic example of creating and navigating the metadata tree."""
    # Initialize a client and metadata tree
    client = DuckDBClient()
    client.connect(auth_method="path", database="metadata.duckdb")
    
    metadata = Metadata(client)
    metadata.create_tables()
    
    # Add a source
    crm_source = metadata.add_source(
        name="crm",
        type="database",
        system="Salesforce",
        owner="sales-team"
    )
    
    # Add objects to the source
    customers = crm_source.add_object(
        name="customers",
        type="table",
        description="Customer master data"
    )
    
    orders = crm_source.add_object(
        name="orders",
        type="table",
        description="Customer orders"
    )
    
    # Navigate the tree upward from an object
    print(f"Object: {customers.name}")
    print(f"  Source: {customers.source.name}")
    print(f"  Owner: {customers.source.owner}")
    
    # Navigate the tree downward from metadata
    print(f"\nMetadata sources: {[s.name for s in metadata.get_sources()]}")
    print(f"CRM objects: {[o.name for o in crm_source.get_objects()]}")


def example_multiple_sources() -> None:
    """Example with multiple sources and objects."""
    client = DuckDBClient()
    client.connect(auth_method="path", database="metadata.duckdb")
    
    metadata = Metadata(client)
    metadata.create_tables()
    
    # Create multiple sources
    crm = metadata.add_source(name="crm", type="database")
    erp = metadata.add_source(name="erp", type="api")
    warehouse = metadata.add_source(name="warehouse", type="database")
    
    # Add objects to each source
    crm.add_object(name="customers", type="table")
    crm.add_object(name="orders", type="table")
    
    erp.add_object(name="invoices", type="table")
    erp.add_object(name="line_items", type="table")
    
    warehouse.add_object(name="fact_sales", type="table")
    warehouse.add_object(name="dim_customer", type="table")
    
    # Get all objects across all sources
    all_objects = metadata.get_all_objects()
    print(f"Total objects: {len(all_objects)}")
    
    # Filter and iterate
    for source in metadata.get_sources():
        print(f"\n{source.name} ({source.type}):")
        for obj in source.get_objects():
            print(f"  - {obj.name} ({obj.type})")


def example_update_metadata() -> None:
    """Example of updating metadata through the tree."""
    client = DuckDBClient()
    client.connect(auth_method="path", database="metadata.duckdb")
    
    metadata = Metadata(client)
    metadata.create_tables()
    
    # Create source and object
    source = metadata.add_source(name="crm", type="database")
    obj = source.add_object(name="customers", type="table")
    
    # Update through the object itself
    obj.update(
        type="view",
        description="Materialized view of customer data"
    )
    
    # Update through the source itself
    source.update(owner="data-ops", timezone="UTC")
    
    # Retrieve and verify
    updated_obj = source.get_object("customers")
    print(f"Object type: {updated_obj.type}")
    print(f"Description: {updated_obj.description}")
    print(f"Source owner: {source.owner}")


def example_active_and_current_filtering() -> None:
    """Example of filtering active and current objects."""
    client = DuckDBClient()
    client.connect(auth_method="path", database="metadata.duckdb")
    
    metadata = Metadata(client)
    metadata.create_tables()
    
    # Create sources and objects with different statuses
    source = metadata.add_source(name="crm", is_active=True)
    
    active_obj = source.add_object(name="active_table", is_active=True)
    archived_obj = source.add_object(name="archived_table", is_active=False)
    
    # Filter active objects
    active = source.get_active_objects()
    print(f"Active objects in source: {[o.name for o in active]}")
    
    # Mark object as archived
    archived_obj.update(is_active=False, is_current=False)
    
    # Get current objects
    current = source.get_current_objects()
    print(f"Current objects: {[o.name for o in current]}")


def example_load_from_database() -> None:
    """Example of loading an existing metadata tree from database."""
    # First, create and populate a database
    client1 = DuckDBClient()
    client1.connect(auth_method="path", database="metadata.duckdb")
    
    metadata1 = Metadata(client1)
    metadata1.create_tables()
    
    crm = metadata1.add_source(name="crm", type="database")
    crm.add_object(name="customers", type="table")
    crm.add_object(name="orders", type="table")
    
    # Later, load from the same database with a new client
    client2 = DuckDBClient()
    client2.connect(auth_method="path", database="metadata.duckdb")
    
    metadata2 = Metadata(client2)
    metadata2.load_from_database()
    
    # Entire tree is now available
    print(f"Sources: {[s.name for s in metadata2.get_sources()]}")
    for source in metadata2.get_sources():
        print(f"  {source.name}: {[o.name for o in source.get_objects()]}")


if __name__ == "__main__":
    print("=== Basic Tree Structure ===")
    example_basic_tree_structure()
    
    print("\n\n=== Multiple Sources ===")
    example_multiple_sources()
    
    print("\n\n=== Update Metadata ===")
    example_update_metadata()
    
    print("\n\n=== Active/Current Filtering ===")
    example_active_and_current_filtering()
    
    print("\n\n=== Load from Database ===")
    example_load_from_database()
