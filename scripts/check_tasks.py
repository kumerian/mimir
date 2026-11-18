#!/usr/bin/env python3
"""Repository smoke checks for the documented, hermetic developer tasks."""

from pathlib import Path
import re
import sys


ROOT = Path(__file__).resolve().parents[1]
MAKEFILE = ROOT / "Makefile"
GO_MOD = ROOT / "go.mod"


def main() -> int:
    errors: list[str] = []
    if not MAKEFILE.is_file():
        errors.append("Makefile is missing")
    else:
        makefile = MAKEFILE.read_text(encoding="utf-8")
        for task in ("test", "test-race", "contract", "integration", "lint", "security", "build", "install-smoke", "docs"):
            if not re.search(rf"(?m)^{re.escape(task)}\s*:", makefile):
                errors.append(f"required make task is missing: {task}")
        if not re.search(r"\$\(GO\)\s+test\s+\./\.\.\.", makefile):
            errors.append("default test task must include hermetic `go test ./...`")
    if not GO_MOD.is_file():
        errors.append("go.mod is missing")
    if errors:
        print("\n".join(f"FAIL: {error}" for error in errors), file=sys.stderr)
        return 1
    print("PASS: pinned module and required hermetic task surface are present")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
