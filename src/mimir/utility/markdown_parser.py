"""Parse a Mimir table-definition markdown file into a :class:`TableDefinition`.

Expected markdown format (see ``metadata/metadata/`` for examples)::

    # Table: `<name>`

    ...prose...

    ## Columns

    | Column | Type | Nullable | Default | Primary Key | Description |
    |--------|------|----------|---------|-------------|-------------|
    | col    | type | YES/NO   | value/— | YES/NO      | ...         |

Column values are case-insensitive.  A dash (``—`` or ``-``) in the
Default column is treated as *no default* (``None``).
"""

from __future__ import annotations

import re
from pathlib import Path

# Maps markdown type strings to ColumnType literals.
_TYPE_MAP: dict[str, str] = {
    "string": "string",
    "text": "text",
    "integer": "integer",
    "int": "integer",
    "bigint": "bigint",
    "float": "float",
    "boolean": "boolean",
    "bool": "boolean",
    "date": "date",
    "timestamp": "timestamp",
    "decimal": "decimal",
}

_TABLE_NAME_RE = re.compile(r"^#\s+Table:\s+`([^`]+)`", re.IGNORECASE)


def load_table_definition(path: Path) -> "TableDefinition":
    """Parse *path* and return the corresponding :class:`TableDefinition`.

    Raises :exc:`ValueError` if the file does not contain a recognisable table
    name heading or a columns table.

    Schema imports are deferred to this function to avoid a circular import
    between ``mimir.utility`` and ``mimir.metadata``.
    """
    from mimir.metadata.schema import ColumnDefinition, TableDefinition  # noqa: PLC0415

    text = path.read_text(encoding="utf-8")
    lines = text.splitlines()

    table_name = _parse_table_name(lines, path)
    columns = _parse_columns(lines, path, ColumnDefinition)

    return TableDefinition(name=table_name, columns=tuple(columns))


# ---------------------------------------------------------------------------
# Internal helpers
# ---------------------------------------------------------------------------


def _parse_table_name(lines: list[str], path: Path) -> str:
    for line in lines:
        match = _TABLE_NAME_RE.match(line.strip())
        if match:
            return match.group(1)
    raise ValueError(
        f"No table name found in '{path}'. "
        "Expected a heading like: # Table: `<name>`"
    )


def _parse_columns(lines: list[str], path: Path, ColumnDefinition: type) -> list:
    """Extract column definitions from the first markdown pipe table found."""
    table_lines = [line.strip() for line in lines if line.strip().startswith("|")]
    if not table_lines:
        raise ValueError(f"No markdown table found in '{path}'.")

    # First row is the header; second row is the separator (|---|...|).
    header_row = table_lines[0]
    headers = _split_row(header_row)
    header_index = {h.lower(): i for i, h in enumerate(headers)}

    required = {"column", "type", "nullable", "primary key"}
    missing = required - header_index.keys()
    if missing:
        raise ValueError(
            f"Columns table in '{path}' is missing required header(s): {sorted(missing)}"
        )

    columns: list[ColumnDefinition] = []
    for row in table_lines[2:]:  # skip header and separator
        cells = _split_row(row)
        if not cells:
            continue

        def cell(header: str, _cells: list[str] = cells) -> str:
            idx = header_index.get(header)
            if idx is None or idx >= len(_cells):
                return ""
            return _cells[idx].strip()

        name = cell("column")
        raw_type = cell("type").lower()
        nullable_str = cell("nullable").upper()
        default_str = cell("default").strip()
        pk_str = cell("primary key").upper()

        if not name:
            continue

        column_type = _TYPE_MAP.get(raw_type)
        if column_type is None:
            raise ValueError(
                f"Unknown column type '{raw_type}' for column '{name}' in '{path}'. "
                f"Supported types: {sorted(_TYPE_MAP)}"
            )

        nullable = nullable_str != "NO"
        primary_key = pk_str == "YES"
        default = None if default_str in {"—", "-", ""} else default_str

        columns.append(
            ColumnDefinition(
                name=name,
                column_type=column_type,
                nullable=nullable,
                primary_key=primary_key,
                default=default,
            )
        )

    if not columns:
        raise ValueError(f"No data rows found in the columns table in '{path}'.")

    return columns


def _split_row(row: str) -> list[str]:
    """Split a markdown pipe-delimited row into a list of cell strings."""
    parts = row.split("|")
    # Strip the empty strings created by leading/trailing pipes.
    return [p.strip() for p in parts[1:-1]]
