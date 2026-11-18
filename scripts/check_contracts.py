#!/usr/bin/env python3
"""Check JSON contract syntax and schema/parser version consistency."""

import json
from pathlib import Path
import re
import sys


ROOT = Path(__file__).resolve().parents[1]
SCHEMA = ROOT / "contracts/metadata/v1.schema.json"
EMBEDDED_SCHEMA = ROOT / "internal/metadata/v1.schema.json"
EXAMPLE = ROOT / "contracts/examples/metadata-v1-draft.json"
SOURCE = ROOT / "internal/metadata/document.go"


def main() -> int:
    try:
        schema = json.loads(SCHEMA.read_text(encoding="utf-8"))
        example = json.loads(EXAMPLE.read_text(encoding="utf-8"))
        source = SOURCE.read_text(encoding="utf-8")
        embedded_schema = EMBEDDED_SCHEMA.read_bytes()
        schema_bytes = SCHEMA.read_bytes()
    except (OSError, json.JSONDecodeError) as error:
        print(f"FAIL: contract syntax or input: {error}", file=sys.stderr)
        return 1
    version_match = re.search(r'ContractVersion\s*=\s*"(mimir\.metadata/v[0-9]+)"', source)
    schema_version = schema.get("properties", {}).get("schemaVersion", {}).get("const")
    if not version_match or schema_version != version_match.group(1) or example.get("schemaVersion") != schema_version:
        print("FAIL: schema, example, and parser metadata versions disagree", file=sys.stderr)
        return 1
    if schema.get("additionalProperties") is not False:
        print("FAIL: metadata schema must reject unknown root fields", file=sys.stderr)
        return 1
    if embedded_schema != schema_bytes:
        print("FAIL: embedded metadata schema is stale; update it from the contract source", file=sys.stderr)
        return 1
    print(f"PASS: JSON syntax and metadata version {schema_version} consistency")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
