# Copilot Instructions for Mimir

## Project Intent
- Build a Python library for metadata-driven ingestion workflows.
- Keep public APIs focused and backwards-compatible when possible.
- Prefer deterministic behavior and simple, testable abstractions.

## Coding Guidelines
- Place library code in `src/mimir`.
- Keep DuckDB interactions isolated in library helpers.
- Add or update tests under `tests/unit` and `tests/integration` for behavior changes.
- Use explicit types for public function signatures.
- Always use the standards as set forth in the `documentation/standards` folder; pep8.rst for python code, SQLstandards.MD for SQL code.

## Testing Expectations
- Unit tests should avoid filesystem and DB side effects when possible.
- Integration tests should use temporary DuckDB files.
