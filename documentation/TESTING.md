# Testing Documentation

This document describes the test structure for the Mimir project and how to run tests.

## Test Structure

The Mimir project uses pytest for testing and is organized into two categories:

### Unit Tests (`tests/unit/`)

Fast, isolated tests that verify individual components without external dependencies.

#### Available Unit Tests

- **`test_database_utilities_instantiation.py`**
  - Verifies that all database client classes can be instantiated without errors
  - Tests: `test_all_database_utilities_can_be_instantiated_empty`
  - Validates: DuckDBClient, MSSQLClient, PostgreSQLClient, MariaDBClient, MySQLClient, OracleClient

- **`test_database_utility_base.py`**
  - Tests the `DatabaseClient` base class functionality
  - Tests:
    - `test_empty_instance_is_not_connected` - Validates initial connection state
    - `test_read_table_requires_connection` - Ensures read operations require connection
    - `test_write_table_requires_connection` - Ensures write operations require connection
    - `test_read_table_returns_dict_rows` - Validates that read operations return rows as dictionaries
    - `test_write_table_appends_rows_and_commits` - Validates insert and commit behavior
    - `test_write_table_replace_deletes_existing_rows` - Validates replace mode functionality

### Integration Tests (`tests/integration/`)

Tests that verify behavior across multiple components or with real database systems.

#### Available Integration Tests

- **`test_duckdb_smoke.py`**
  - Smoke test for DuckDB connectivity
  - Tests: `test_duckdb_temp_database_round_trip`
  - Validates: Basic DuckDB connection and query execution with temporary database files

## Running Tests

### Prerequisites

Install development dependencies:

```bash
python -m pip install -e .[dev]
```

This installs:
- pytest (test framework)
- pytest-cov (coverage reporting)
- ruff (linting)

### Run All Tests

Execute all unit and integration tests:

```bash
pytest
```

### Run Specific Test Categories

Run only unit tests:

```bash
pytest -m unit
```

Run only integration tests:

```bash
pytest -m integration
```

### Run a Specific Test File

```bash
pytest tests/unit/test_database_utilities_instantiation.py
```

```bash
pytest tests/integration/test_duckdb_smoke.py
```

### Run a Specific Test

```bash
pytest tests/unit/test_database_utilities_instantiation.py::test_all_database_utilities_can_be_instantiated_empty
```

### Run Tests with Coverage Report

```bash
pytest --cov=mimir --cov-report=html
```

This generates an HTML coverage report in `htmlcov/index.html`.

### Run Tests with Verbose Output

```bash
pytest -v
```

Shows detailed test names and results for each test.

### Run Tests with Summary Report

```bash
pytest -ra
```

Shows a summary of all test results (pass, fail, xfail, etc.).

## Test Configuration

Test configuration is defined in `pyproject.toml` under `[tool.pytest.ini_options]`:

- `testpaths = ["tests"]` - Tests are located in the `tests` directory
- `addopts = "-ra"` - Default to showing summary report
- `xfail_strict = true` - Treats expected failures as errors
- Markers:
  - `integration` - Marks integration tests
  - `unit` - Marks unit tests

## Best Practices

- Unit tests should avoid filesystem and database side effects when possible
- Integration tests use temporary DuckDB files (via pytest's `tmp_path` fixture)
- All tests are marked with appropriate markers (`@pytest.mark.unit` or `@pytest.mark.integration`)
- Use explicit assertions for clarity
- Mock external dependencies in unit tests
