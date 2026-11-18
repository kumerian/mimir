#!/usr/bin/env python3
"""Fail if any Go source file is not gofmt-formatted."""

from pathlib import Path
import subprocess
import sys


ROOT = Path(__file__).resolve().parents[1]
files = sorted(str(path) for path in ROOT.rglob("*.go") if ".git" not in path.parts)
result = subprocess.run(["gofmt", "-l", *files], cwd=ROOT, text=True, capture_output=True, check=False)
if result.returncode:
    print(result.stderr, file=sys.stderr, end="")
    raise SystemExit(result.returncode)
if result.stdout.strip():
    print("Go formatting required for:\n" + result.stdout, file=sys.stderr, end="")
    raise SystemExit(1)
print("PASS: Go formatting")
