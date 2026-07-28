# Mimir Agent Workflow Guide

This guide explains how to use the Mimir project agents and their handoff workflow.

## Overview

The Mimir project uses three specialized agents that work together to develop, test, and document features:

```
Developer → Test Writer → Documentation Agent
    ↑          ↓
    └──────────┘ (if tests fail)
```

## The Three Agents

### 1. 🛠️ Mimir Python Developer
- **Focus**: Writing clean, well-typed Python code
- **Expertise**: Implementation, refactoring, bug fixes
- **Standards**: PEP 8, type hints, DuckDB isolation
- **Files**: `.python-developer.agent.md`

### 2. ✅ Mimir Test Writer
- **Focus**: Writing comprehensive pytest tests
- **Expertise**: Unit tests, integration tests, test validation
- **Standards**: Test markers, mocking, isolation
- **Files**: `.test-writer.agent.md`

### 3. 📚 Mimir Documentation Agent
- **Focus**: Keeping documentation synchronized with code
- **Expertise**: Docstrings, API docs, guides, README updates
- **Standards**: Clear language, examples, completeness
- **Files**: `.documentation.agent.md`

## Workflow Stages

### Stage 1: Implementation (Developer)

**Invoke**: `@Mimir Python Developer`

**Tasks**:
- Implement new features or fix bugs
- Write clean, typed Python code following PEP 8
- Add docstrings for public APIs
- Test locally with `pytest`

**Handoff**: When implementation is complete, say:
> "Implementation complete. Handing to @Mimir Test Writer for comprehensive testing."

### Stage 2: Testing (Test Writer)

**Invoke**: `@Mimir Test Writer` (receives handoff from Developer)

**Tasks**:
- Write unit tests for new code
- Write integration tests where needed
- Use pytest markers (`@pytest.mark.unit`, `@pytest.mark.integration`)
- Mock external dependencies in unit tests
- Use temporary files in integration tests

**Two Paths**:

**Path A - Tests Pass** ✓
> "All tests passed successfully! Handing to @Mimir Documentation Agent for documentation updates."
→ Move to Stage 3

**Path B - Tests Fail** ✗
> "Tests failed. [Specific failures]. Handing back to @Mimir Python Developer for fixes."
→ Return to Stage 1

### Stage 3: Documentation (Documentation Agent)

**Invoke**: `@Mimir Documentation Agent` (receives handoff from Test Writer)

**Tasks**:
- Update code docstrings if needed
- Update README.md for user-facing changes
- Update testing documentation if new patterns introduced
- Create guides for complex features
- Verify API documentation is current

**Completion**:
> "Documentation complete! Feature is fully implemented, tested, and documented. ✓"

## How to Use These Agents

### Option 1: Use Individual Agents for Specific Tasks

If you have a specific task, invoke the appropriate agent directly:

```
You: @Mimir Python Developer, implement feature X
Developer: [implements code]
You: @Mimir Test Writer, write tests for this feature
Test Writer: [writes tests]
```

### Option 2: Use the Full Workflow

Start with the Developer and let agents hand off automatically:

```
You: @Mimir Python Developer, implement feature X with full workflow
Developer: [implements] → hands to Test Writer
Test Writer: [tests] → hands to Documentation Agent
Documentation Agent: [updates docs] → Complete!
```

### Option 3: Multi-agent Collaboration

For complex features, multiple agents can work together:

```
You: @Mimir Python Developer, start implementing the new ingestion API

Developer: [creates implementation]
You: @Mimir Test Writer, test the ingestion API
Test Writer: [finds issues] → hands back to Developer

Developer: [fixes issues] → hands to Test Writer
Test Writer: [all tests pass] → hands to Documentation Agent

Documentation Agent: [updates docs] → Complete!
```

## Useful Commands for Each Agent

### Developer Commands
```bash
# Test locally before handoff
pytest

# Test specific file
pytest tests/unit/test_filename.py

# Check code style
ruff check src/
```

### Test Writer Commands
```bash
# Run all tests
pytest

# Run only unit tests (fast)
pytest -m unit

# Run only integration tests
pytest -m integration

# Check coverage
pytest --cov=mimir --cov-report=html
```

### Documentation Agent Commands
```bash
# Verify links work (manual)
# Review file structure
ls documentation/
ls tests/
```

## Best Practices for Agent Workflows

### For Developers

1. **Write focused code**: Each change should do one thing well
2. **Include docstrings**: Document public APIs clearly
3. **Test locally**: Run `pytest` before handing to Test Writer
4. **Follow standards**: Use PEP 8 and type hints consistently
5. **Provide context**: Explain what you implemented and why

### For Test Writers

1. **Read the code**: Understand implementation details
2. **Write comprehensive tests**: Cover happy paths and edge cases
3. **Test locally**: Run `pytest` before handing off
4. **Provide detailed feedback**: If tests fail, explain specifically what's wrong
5. **Check coverage**: Ensure adequate test coverage

### For Documentation Specialists

1. **Review the code**: Understand the feature and tests
2. **Write clearly**: Use simple language with good examples
3. **Keep consistency**: Match existing documentation style
4. **Update proactively**: Find all places that need updates
5. **Verify examples**: Test that code examples actually work

## Example Workflow: Adding a New Database Client

1. **Developer**:
   - Implements `PostgreSQLv2Client` class
   - Adds type hints and docstrings
   - Tests locally with `pytest -m unit`
   - Hands to Test Writer

2. **Test Writer**:
   - Writes unit tests for `PostgreSQLv2Client`
   - Writes integration test with temporary database
   - Runs `pytest` and checks coverage
   - Tests pass → Hands to Documentation Agent
   - Tests fail → Hands back to Developer

3. **Documentation Agent**:
   - Updates README with new client info
   - Adds comprehensive docstrings
   - Updates API documentation
   - Marks feature as complete ✓

## Troubleshooting Agent Handoffs

### Developer → Test Writer Handoff Issues

**Problem**: Test Writer receives code but it's not testable
**Solution**: Developer should test locally first with `pytest`

**Problem**: Missing docstrings or type hints
**Solution**: Developer should add these before handoff

### Test Writer → Developer Handoff Issues

**Problem**: Developer doesn't understand why tests failed
**Solution**: Test Writer should provide detailed failure analysis with error messages

**Problem**: Developer receives feedback but can't reproduce locally
**Solution**: Test Writer should provide exact command to reproduce

### Test Writer → Documentation Handoff Issues

**Problem**: Documentation Agent doesn't know what changed
**Solution**: Test Writer should summarize the feature and point to new/changed code

## Quick Reference

| Agent | Input | Output | Handoff To |
|-------|-------|--------|-----------|
| Developer | Feature request/bug report | Implementation + tests locally passing | Test Writer |
| Test Writer | Implementation | Tests (passed or failed) | Dev (fail) or Docs (pass) |
| Documentation | Tested code | Updated docs + docstrings | Done! |

## Related Files

- [AGENTS.md](AGENTS.md) - Detailed agent definitions
- [Testing Documentation](documentation/TESTING.md) - How to run tests
- [Python Developer Agent](`.python-developer.agent.md`)
- [Test Writer Agent](`.test-writer.agent.md`)
- [Documentation Agent](`.documentation.agent.md`)
- [README](README.md)
