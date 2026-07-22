# Mímir

Create your own Mímisbrunnr. Your well of wisdom. Mímir will guard your well of wisdom by automating the ingestion of all of your wisdom. 
 Well, your data but you'll get some wisdom from that, right?

Mímir is a Python library for metadata-driven data ingestion tooling.

## Project Structure

- `src/mimir`: library source code
- `tests/unit`: fast unit test suite
- `tests/integration`: DuckDB integration test suite
- `data/reference.duckdb`: default reference database path
- `sql/reference_schema.sql`: DuckDB schema used by the library
- `.github/workflows/ci.yml`: GitHub Actions pipeline
- `.github/copilot-instructions.md`: repository Copilot coding guidance
- `.github/chatmodes/mimir-maintainer.agent.md`: Copilot agent definition
- `.github/skills/metadata-ingestion/SKILL.md`: Copilot skill definition
- `docs`: project and library documentation
- `.github/ISSUE_TEMPLATE`: GitHub issue templates

## Quick Start

1. Create and activate a virtual environment.
1. Install the package and development dependencies:

```bash
pip install -e .[dev]
```

1. Bootstrap the reference DuckDB database:

```bash
python scripts/bootstrap_reference_db.py
```

## Running Tests

Unit tests:

```bash
pytest tests/unit -m unit
```

Integration tests:

```bash
pytest tests/integration -m integration
```

All tests:

```bash
pytest tests
```

## Documentation

Start with `docs/README.md` for links to detailed library and testing guides.
