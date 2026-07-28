"""Architecture comparison and migration guide for the new hierarchical metadata structure.

This document explains the transition from the repository-based approach
to the new hierarchical tree-based approach.
"""

# ============================================================================
# BEFORE: Repository Pattern (Legacy)
# ============================================================================
#
# The old approach used separate repository classes for each entity type:
#
#   ObjectRepository  ─┐
#                      ├─ DatabaseClient
#   SourceRepository  ─┘
#
# Problems with the repository approach:
# - No parent-child relationships in the object model
# - Duplicate database client references
# - Hard to navigate from object to source to database
# - No natural way to express hierarchical queries
# - Inconsistent state between memory and database
#
# Example (OLD):
#   repo = ObjectRepository(client)
#   records = repo.get_by_source("crm")
#   # No way to know which source owns these objects
#   # No parent reference back to source
#
# ============================================================================
# AFTER: Hierarchical Tree Structure (NEW)
# ============================================================================
#
# The new approach uses a tree structure with parent-child relationships:
#
#                     ┌─── Metadata (root)
#                     │        │
#                     │      ──┴──
#                    │      │     │
#                  Source  Source Source
#                    │       │      │
#                   ─┴─     ─┴─    ─┴─
#                  │  │   │  │   │   │
#                 Obj Obj Obj Obj Obj Obj
#
# Advantages of the tree approach:
# ✓ Natural parent-child relationships
# ✓ Single client connection shared through tree
# ✓ Easy navigation in both directions (up and down)
# ✓ Intuitive hierarchical queries
# ✓ Single source of truth for state
# ✓ Closer to domain model
#
# Example (NEW):
#   metadata = Metadata(client)
#   metadata.create_tables()
#   
#   crm = metadata.add_source(name="crm")
#   customers = crm.add_object(name="customers")
#
#   # Navigate upward
#   print(customers.source.name)  # "crm"
#   print(customers.source.metadata)  # <Metadata object>
#
#   # Navigate downward
#   for source in metadata.get_sources():
#       for obj in source.get_objects():
#           print(f"{source.name}.{obj.name}")
#
# ============================================================================
# API COMPARISON
# ============================================================================

## Creating metadata

# OLD (Repository Pattern)
# source_repo = SourceRepository(client)
# source_repo.create_table()
# source_repo.insert(SourceRecord(name="crm", type="database"))
# 
# object_repo = ObjectRepository(client)
# object_repo.create_table()
# object_repo.insert(ObjectRecord(name="customers", source_name="crm"))

# NEW (Hierarchical Tree)
# metadata = Metadata(client)
# metadata.create_tables()
# 
# source = metadata.add_source(name="crm", type="database")
# obj = source.add_object(name="customers")


## Reading metadata

# OLD (Repository Pattern)
# sources = source_repo.get_all()
# objects = object_repo.get_by_source("crm")
# # No connection between objects and their source

# NEW (Hierarchical Tree)
# sources = metadata.get_sources()
# crm = metadata.get_source("crm")
# objects = crm.get_objects()
# # Objects know their source: objects[0].source


## Updating metadata

# OLD (Repository Pattern)
# source_repo.update("crm", owner="data-team")
# object_repo.update("customers", type="view")

# NEW (Hierarchical Tree)
# source = metadata.get_source("crm")
# source.update(owner="data-team")
# 
# obj = source.get_object("customers")
# obj.update(type="view")


## Querying with filters

# OLD (Repository Pattern)
# active_sources = source_repo.get_active()
# current_objects = object_repo.get_current()
# # No way to efficiently get all objects across all sources

# NEW (Hierarchical Tree)
# active_sources = metadata.get_active_sources()
# current_objects = metadata.get_all_objects()  # or filter
# 
# # Hierarchical queries
# for source in metadata.get_active_sources():
#     for obj in source.get_active_objects():
#         process(obj)


## Loading existing data

# OLD (Repository Pattern)
# source_repo.get_all()  # Returns SourceRecords
# object_repo.get_all()  # Returns ObjectRecords
# # No automatic reconstruction of relationships

# NEW (Hierarchical Tree)
# metadata = Metadata(client)
# metadata.load_from_database()
# # Entire tree is reconstructed with all relationships


# ============================================================================
# MIGRATION PATH
# ============================================================================
#
# The new implementation is FULLY BACKWARD COMPATIBLE:
#
# 1. Existing code using ObjectRepository and SourceRepository continues to work
# 2. Both APIs are available simultaneously
# 3. New code should use the Metadata/Source/Object classes
# 4. Existing code can migrate gradually
#
# Migration strategy:
#   Phase 1: Keep both APIs running (current state)
#   Phase 2: New features use hierarchical API
#   Phase 3: Gradually refactor existing code to use hierarchical API
#   Phase 4: Deprecate and remove repository classes (future)
#

# ============================================================================
# CLASS REFERENCE
# ============================================================================

## Metadata (root node)

# Attributes:
#   - _client: DatabaseClient
#   - sources: dict[str, Source]
#
# Methods:
#   - create_tables(if_not_exists=True) -> tuple[str, str]
#   - add_source(name, type=None, ...) -> Source
#   - get_source(name) -> Source | None
#   - get_sources() -> list[Source]
#   - get_active_sources() -> list[Source]
#   - get_current_sources() -> list[Source]
#   - get_all_objects() -> list[Object]
#   - load_from_database() -> None
#   - _update_source(name, **fields) -> bool
#   - _update_object(name, **fields) -> bool


## Source (intermediate node)

# Attributes:
#   - name: str
#   - metadata: Metadata
#   - type: str | None
#   - system: str | None
#   - owner: str | None
#   - timezone: str | None
#   - is_active: bool
#   - is_current: bool
#   - objects: dict[str, Object]
#
# Methods:
#   - to_dict() -> dict[str, Any]
#   - add_object(name, type=None, ...) -> Object
#   - get_object(name) -> Object | None
#   - get_objects() -> list[Object]
#   - get_active_objects() -> list[Object]
#   - get_current_objects() -> list[Object]
#   - update(**fields) -> bool
#   - delete() -> None
#   - _load_objects_from_db() -> None


## Object (leaf node)

# Attributes:
#   - name: str
#   - source: Source
#   - type: str | None
#   - description: str | None
#   - is_active: bool
#   - is_current: bool
#
# Methods:
#   - to_dict() -> dict[str, Any]
#   - update(**fields) -> bool
#   - delete() -> None


# ============================================================================
# DESIGN PRINCIPLES
# ============================================================================
#
# 1. Single Responsibility:
#    - Metadata manages the database and sources
#    - Source manages objects within itself
#    - Object represents metadata for a single entity
#
# 2. Parent-Child Relationships:
#    - Each child maintains a reference to its parent
#    - Enables upward navigation (obj→source→metadata)
#    - Parent can query/manage children
#
# 3. Lazy Loading:
#    - Objects are loaded on-demand from database
#    - In-memory cache is maintained in parent
#    - load_from_database() loads everything
#
# 4. Consistency:
#    - Updates are immediately persisted to database
#    - In-memory state matches database state
#    - Bulk operations maintain invariants
#
# 5. Type Safety:
#    - Full type hints for all public APIs
#    - Dataclass-based records for clarity
#    - Explicit error messages with ValueError/etc
#

print(__doc__)
