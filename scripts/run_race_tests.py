#!/usr/bin/env python3
"""Run race-enabled Go tests after checking their C compiler prerequisite."""

import argparse
import os
from pathlib import Path
import shutil
import subprocess
import sys


ROOT = Path(__file__).resolve().parents[1]


def find_compiler(go: str, env: dict[str, str]) -> str | None:
    result = subprocess.run(
        [go, "env", "CC"],
        cwd=ROOT,
        env=env,
        text=True,
        capture_output=True,
    )
    if result.returncode != 0:
        raise RuntimeError(result.stderr.strip() or "go env CC failed")
    compiler = result.stdout.strip()
    if not compiler:
        return None
    return shutil.which(compiler) or (str(Path(compiler)) if Path(compiler).is_file() else None)


def run(go: str) -> int:
    env = os.environ.copy()
    try:
        compiler = find_compiler(go, env)
    except (OSError, RuntimeError) as error:
        print(f"FAIL: cannot inspect the Go race-test environment: {error}", file=sys.stderr)
        return 2
    if compiler is None:
        print(
            "FAIL: race tests require a C compiler, but none is available. "
            "Install the compiler documented in docs/operations/local-development.md "
            "and make it available on PATH, then rerun make test-race.",
            file=sys.stderr,
        )
        return 2
    env["CGO_ENABLED"] = "1"
    try:
        return subprocess.run([go, "test", "-race", "./..."], cwd=ROOT, env=env, check=False).returncode
    except OSError as error:
        print(f"FAIL: cannot start Go race tests: {error}", file=sys.stderr)
        return 2


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--go", default="go", help="Go executable to use")
    args = parser.parse_args()
    return run(args.go)


if __name__ == "__main__":
    raise SystemExit(main())
