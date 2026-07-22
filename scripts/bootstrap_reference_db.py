"""Bootstrap script for creating the local reference DuckDB database."""

from __future__ import annotations

import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SRC = ROOT / "src"

if str(SRC) not in sys.path:
    sys.path.insert(0, str(SRC))

from src.mimir.library import ensure_reference_database


if __name__ == "__main__":
    path = ensure_reference_database()
    print(f"Reference database initialized at: {path}")
