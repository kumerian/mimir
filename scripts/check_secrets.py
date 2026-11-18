#!/usr/bin/env python3
"""Fail on a small set of high-confidence committed credential patterns."""

from pathlib import Path
import re
import sys


ROOT = Path(__file__).resolve().parents[1]
PATTERNS = (
    re.compile(r"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----"),
    re.compile(r"\bAKIA[0-9A-Z]{16}\b"),
    re.compile(r"\bgh[pousr]_[A-Za-z0-9]{30,}\b"),
)
SKIP = {".git", "bin", "dist"}


def main() -> int:
    errors = []
    for path in ROOT.rglob("*"):
        if not path.is_file() or any(part in SKIP for part in path.parts):
            continue
        try:
            text = path.read_text(encoding="utf-8")
        except (UnicodeDecodeError, OSError):
            continue
        for pattern in PATTERNS:
            if pattern.search(text):
                errors.append(f"{path.relative_to(ROOT)} matches high-confidence secret pattern {pattern.pattern}")
    if errors:
        print("\n".join(f"FAIL: {error}" for error in errors), file=sys.stderr)
        return 1
    print("PASS: no high-confidence credential patterns found")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
