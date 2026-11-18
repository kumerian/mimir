#!/usr/bin/env python3
"""Run an explicitly classified Go test suite or report its scope as N/A."""

from pathlib import Path
import subprocess
import sys


ROOT = Path(__file__).resolve().parents[1]


def main() -> int:
    if len(sys.argv) != 2 or sys.argv[1] not in {"integration", "acceptance", "production_smoke"}:
        print("usage: run_tagged.py integration|acceptance|production_smoke", file=sys.stderr)
        return 2
    tag = sys.argv[1]
    matches = [
        path for path in ROOT.rglob("*.go")
        if ".git" not in path.parts and f"//go:build {tag}" in path.read_text(encoding="utf-8")
    ]
    if not matches:
        print(f"NOT APPLICABLE: no Go files declare the {tag!r} build tag; no external environment or credentials are provisioned.")
        return 0
    return subprocess.run(["go", "test", f"-tags={tag}", "./..."], cwd=ROOT, check=False).returncode


if __name__ == "__main__":
    raise SystemExit(main())
