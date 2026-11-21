#!/usr/bin/env python3
"""Bind a local GUI candidate to its source revision, binary, and frontend files."""

import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import sys


ROOT = Path(__file__).resolve().parents[1]


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as source:
        for chunk in iter(lambda: source.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def frontend_identity(dist: Path) -> dict[str, object]:
    if not dist.is_dir():
        raise ValueError("frontend dist directory is unavailable")
    files = sorted(path for path in dist.rglob("*") if path.is_file())
    if not files:
        raise ValueError("frontend dist contains no files")

    digest = hashlib.sha256()
    file_entries: list[dict[str, object]] = []
    for path in files:
        relative = path.relative_to(dist).as_posix()
        file_digest = sha256_file(path)
        file_entries.append({"path": relative, "sha256": file_digest})
        digest.update(relative.encode("utf-8"))
        digest.update(b"\0")
        digest.update(bytes.fromhex(file_digest))
    return {
        "sha256": digest.hexdigest(),
        "fileCount": len(file_entries),
        "files": file_entries,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", required=True, type=Path)
    parser.add_argument("--frontend-dist", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--require-clean", action="store_true")
    args = parser.parse_args()

    try:
        revision = subprocess.run(
            ["git", "rev-parse", "HEAD"], cwd=ROOT, check=True, text=True, capture_output=True
        ).stdout.strip()
        status = subprocess.run(
            ["git", "status", "--porcelain", "--untracked-files=all"],
            cwd=ROOT,
            check=True,
            text=True,
            capture_output=True,
        ).stdout
        if args.require_clean and status:
            raise ValueError("candidate source checkout is not clean")
        if not args.binary.is_file():
            raise ValueError("candidate binary is unavailable")

        health = subprocess.run(
            [str(args.binary.resolve()), "health"],
            check=False,
            text=True,
            capture_output=True,
        )
        if health.returncode:
            raise ValueError("candidate health command failed")
        health_json = json.loads(health.stdout)
        contracts = health_json.get("contracts")
        if not isinstance(contracts, list) or not contracts:
            raise ValueError("candidate health did not report contracts")

        identity = {
            "sourceRevision": revision,
            "workingTreeDirty": bool(status),
            "workingTreeStatusSha256": hashlib.sha256(status.encode()).hexdigest(),
            "candidateEligible": not bool(status),
            "binarySha256": sha256_file(args.binary),
            "frontend": frontend_identity(args.frontend_dist),
            "contractVersions": sorted(contracts),
        }
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(identity, sort_keys=True, indent=2) + "\n", encoding="utf-8")
    except (OSError, ValueError, json.JSONDecodeError, subprocess.CalledProcessError) as error:
        print(f"FAIL: GUI candidate identity: {error}", file=sys.stderr)
        return 1

    print(f"PASS: GUI candidate identity for {revision}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
