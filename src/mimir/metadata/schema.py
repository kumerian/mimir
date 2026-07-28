"""Database-agnostic schema models used by metadata operations."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Literal

ColumnType = Literal[
    "string",
    "text",
    "integer",
    "bigint",
    "float",
    "boolean",
    "date",
    "timestamp",
    "decimal",
]


@dataclass(frozen=True)
class ColumnDefinition:
    """Logical column definition used across all supported databases."""

    name: str
    column_type: ColumnType
    nullable: bool = True
    primary_key: bool = False
    default: str | None = None
    length: int | None = None
    precision: int | None = None
    scale: int | None = None


@dataclass(frozen=True)
class TableDefinition:
    """Logical table definition used across all supported databases."""

    name: str
    columns: tuple[ColumnDefinition, ...]
