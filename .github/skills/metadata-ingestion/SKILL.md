---
name: metadata-ingestion
description: Workflow for changing metadata source registration and DuckDB-backed reference behaviors.
---

# Metadata Ingestion Skill

## When to Use
Use this skill when changes involve:
- Registering or listing metadata sources
- DuckDB reference schema or bootstrap behavior
- Integration tests that validate metadata persistence

## Guardrails
- Keep schema changes backward compatible where possible.
- Add integration tests for all DB behavior changes.
- Document changes in `docs/library.md` when public behavior changes.

## Checklist
1. Update library logic in `src/mimir/library.py`.
2. Update schema in `sql/reference_schema.sql` if needed.
3. Add unit tests for pure logic changes.
4. Add integration tests for DB behavior.
5. Run `pytest tests/unit` and `pytest tests/integration`.
