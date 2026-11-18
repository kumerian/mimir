#!/usr/bin/env python3
"""Copy the canonical metadata schema to the Go embed location."""

from pathlib import Path
import sys


ROOT = Path(__file__).resolve().parents[1]
SOURCE = ROOT / "contracts/metadata/v1.schema.json"
TARGET = ROOT / "internal/metadata/v1.schema.json"


def main() -> int:
    try:
        TARGET.write_bytes(SOURCE.read_bytes())
    except OSError as error:
        print(f"FAIL: unable to generate embedded metadata schema: {error}", file=sys.stderr)
        return 1
    print("PASS: generated embedded metadata schema from canonical contract")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
