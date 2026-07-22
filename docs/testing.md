# Testing Guide

Mimir separates tests into unit and integration suites.

## Unit Tests

Unit tests validate logic without requiring persistent external state.

Command:

- `pytest tests/unit -m unit`

## Integration Tests

Integration tests validate DuckDB behavior using temporary databases.

Command:

- `pytest tests/integration -m integration`

## Full Test Run

- `pytest tests`
