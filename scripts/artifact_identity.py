#!/usr/bin/env python3
"""Create a candidate manifest from the executable's reported contracts."""

import argparse
from hashlib import sha256
import json
from pathlib import Path
import re
import subprocess
import sys


ROOT = Path(__file__).resolve().parents[1]
REVISION_PATTERN = re.compile(r"^[0-9a-f]{40,64}$")


def repository_state(root: Path) -> tuple[str, str]:
    revision = subprocess.run(
        ["git", "rev-parse", "HEAD"], cwd=root, check=True, text=True, capture_output=True
    ).stdout.strip()
    status = subprocess.run(
        ["git", "status", "--porcelain", "--untracked-files=all"],
        cwd=root,
        check=True,
        text=True,
        capture_output=True,
    ).stdout
    return revision, status


def create_identity(binary: Path, source_revision: str, working_tree_status: str) -> dict[str, object]:
    if not binary.is_file():
        raise ValueError("candidate executable is unavailable")
    if not REVISION_PATTERN.fullmatch(source_revision):
        raise ValueError("candidate source revision must be a full Git object ID")

    try:
        result = subprocess.run(
            [str(binary.resolve()), "health"],
            check=False,
            text=True,
            capture_output=True,
        )
    except OSError as error:
        raise RuntimeError("candidate health command could not be started") from error
    if result.returncode:
        raise RuntimeError("candidate health command failed")
    try:
        health = json.loads(result.stdout)
    except (TypeError, json.JSONDecodeError) as error:
        raise ValueError("candidate health output is not valid JSON") from error

    if not isinstance(health, dict):
        raise ValueError("candidate health output is incomplete")
    contracts = health.get("contracts")
    if health.get("status") != "ready" or not isinstance(contracts, list) or not contracts:
        raise ValueError("candidate health output is incomplete")
    if any(not isinstance(contract, str) or not contract.strip() for contract in contracts):
        raise ValueError("candidate health contains an invalid contract identity")
    if len(set(contracts)) != len(contracts):
        raise ValueError("candidate health contains duplicate contract identities")

    return {
        "sourceRevision": source_revision,
        "workingTreeDirty": bool(working_tree_status),
        "workingTreeStatusSha256": sha256(working_tree_status.encode()).hexdigest(),
        "candidateEligible": not bool(working_tree_status),
        "binarySha256": sha256(binary.read_bytes()).hexdigest(),
        "contractVersions": sorted(contracts),
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", required=True, type=Path, help="candidate executable")
    parser.add_argument("--output", required=True, type=Path, help="identity JSON output path")
    parser.add_argument("--root", type=Path, default=ROOT, help="Git checkout root")
    parser.add_argument("--require-clean", action="store_true", help="reject a dirty source checkout")
    args = parser.parse_args()

    try:
        revision, status = repository_state(args.root)
        if args.require_clean and status:
            raise ValueError("candidate source checkout is not clean")
        identity = create_identity(args.binary, revision, status)
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(identity, sort_keys=True, indent=2) + "\n", encoding="utf-8")
    except (OSError, RuntimeError, ValueError, subprocess.CalledProcessError) as error:
        print(f"FAIL: candidate identity: {error}", file=sys.stderr)
        return 1

    print(f"PASS: candidate identity for {revision}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
