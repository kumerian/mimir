#!/usr/bin/env python3
"""Run one required check with an explicit name and failure status."""

import subprocess
import sys


def main(argv: list[str]) -> int:
    if "--" not in argv:
        print("usage: required_check.py CHECK-NAME -- COMMAND [ARG ...]", file=sys.stderr)
        return 2
    separator = argv.index("--")
    name = " ".join(argv[:separator]).strip()
    command = argv[separator + 1 :]
    if not name or not command:
        print("usage: required_check.py CHECK-NAME -- COMMAND [ARG ...]", file=sys.stderr)
        return 2
    try:
        result = subprocess.run(command, check=False)
    except OSError as error:
        print(f"FAIL: required check {name!r} could not start ({error.__class__.__name__})", file=sys.stderr)
        return 127
    if result.returncode:
        code = result.returncode if result.returncode > 0 else 1
        print(f"FAIL: required check {name!r} exited with status {result.returncode}", file=sys.stderr)
        return code
    print(f"PASS: required check {name!r}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
