#!/usr/bin/env python3
"""Fail if any Go source file is not gofmt-formatted."""

import argparse
from pathlib import Path
import subprocess
import sys


ROOT = Path(__file__).resolve().parents[1]
parser = argparse.ArgumentParser()
parser.add_argument("--gofmt", default="gofmt", help="gofmt executable to use")
args = parser.parse_args()
files = sorted(str(path) for path in ROOT.rglob("*.go") if ".git" not in path.parts)
unformatted = []
try:
    for filename in files:
        path = Path(filename)
        source = path.read_text(encoding="utf-8").replace("\r\n", "\n").replace("\r", "\n")
        result = subprocess.run(
            [args.gofmt],
            cwd=ROOT,
            input=source,
            text=True,
            capture_output=True,
            check=False,
        )
        if result.returncode:
            print(f"FAIL: gofmt could not format {path.relative_to(ROOT)}", file=sys.stderr)
            print(result.stderr, file=sys.stderr, end="")
            raise SystemExit(result.returncode)
        if result.stdout != source:
            unformatted.append(str(path.relative_to(ROOT)))
except OSError as error:
    print(f"FAIL: cannot start gofmt: {error}", file=sys.stderr)
    raise SystemExit(2)
if unformatted:
    print("Go formatting required for:\n" + "\n".join(unformatted) + "\n", file=sys.stderr, end="")
    raise SystemExit(1)
print("PASS: Go formatting")
