# Mimir

Create your own Mímisbrunnr. Your well of wisdom. Mímir will guard your well of wisdom by automating the ingestion of all of your wisdom. 
 Well, your data but you'll get some wisdom from that, right?

Mímir is a Python library for metadata-driven data ingestion tooling.

## Development

1. Create and activate a virtual environment.
2. Install dev dependencies:

```bash
python -m pip install -e .[dev]
```

3. Run tests:

```bash
pytest
```

## Agents for Development

This project includes specialized agents to streamline development:

- **🛠️ Mimir Python Developer** - Writes and implements Python code
- **✅ Mimir Test Writer** - Creates and validates tests
- **📚 Mimir Documentation Agent** - Updates and maintains documentation

Agents work together in a handoff workflow:
1. Developer implements code → hands to Test Writer
2. Test Writer validates with tests → hands to Documentation Agent (if pass) or back to Developer (if fail)
3. Documentation Agent updates docs → feature complete

See [Agent Workflow Guide](documentation/AGENT_WORKFLOW.md) for details on using the agents.

## Project Layout

- `src/mimir`: library source code
- `tests/unit`: fast unit tests
- `tests/integration`: integration tests
- `documentation/`: guides and standards
  - `TESTING.md`: test documentation
  - `AGENT_WORKFLOW.md`: how to use the development agents
  - `standards/`: PEP 8 and SQL standards
