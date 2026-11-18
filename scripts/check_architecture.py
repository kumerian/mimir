#!/usr/bin/env python3
"""Reject dependency-direction violations in the Go package layout."""

from pathlib import Path
import re
import sys


ROOT = Path(__file__).resolve().parents[1]
FORBIDDEN = {
    "internal/domain": ("internal/local", "internal/adapters", "internal/entrypoints", "cmd/"),
    "internal/metadata": ("internal/local", "internal/adapters", "internal/entrypoints", "cmd/"),
    "internal/planning": ("internal/local", "internal/adapters", "internal/entrypoints", "cmd/"),
    "internal/state": ("internal/local", "internal/adapters", "internal/entrypoints", "cmd/"),
    "internal/scheduler": ("internal/local", "internal/adapters", "internal/entrypoints", "cmd/"),
    "internal/ports": ("internal/local", "internal/adapters", "internal/entrypoints", "cmd/"),
    "internal/application": ("internal/local", "internal/adapters", "internal/entrypoints", "cmd/"),
}


def dependency_violations(root: Path, rules=FORBIDDEN) -> list[str]:
    errors = []
    for package, forbidden in rules.items():
        directory = root / package
        if not directory.exists():
            continue
        for path in directory.rglob("*.go"):
            text = path.read_text(encoding="utf-8")
            for import_path in re.findall(r'"([^"]+)"', text):
                if any(marker in import_path for marker in forbidden):
                    errors.append(f"{path.relative_to(root)} imports forbidden dependency {import_path}")
    return errors


def main() -> int:
    errors = dependency_violations(ROOT)
    if errors:
        print("\n".join(f"FAIL: {error}" for error in errors), file=sys.stderr)
        return 1
    print("PASS: checked domain, ports, planning, state, and scheduler dependency direction")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
