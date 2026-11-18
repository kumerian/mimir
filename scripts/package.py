#!/usr/bin/env python3
"""Create a local executable archive with an identity manifest."""

from hashlib import sha256
from pathlib import Path
import gzip
import json
import subprocess
import tarfile


ROOT = Path(__file__).resolve().parents[1]
DIST = ROOT / "dist"
BIN = ROOT / "bin/mimir"


def main() -> int:
    DIST.mkdir(exist_ok=True)
    revision = subprocess.run(
        ["git", "rev-parse", "HEAD"], cwd=ROOT, check=True, text=True, capture_output=True
    ).stdout.strip()
    status = subprocess.run(
        ["git", "status", "--porcelain", "--untracked-files=all"],
        cwd=ROOT, check=True, text=True, capture_output=True,
    ).stdout
    digest = sha256(BIN.read_bytes()).hexdigest()
    manifest = {
        "sourceRevision": revision,
        "workingTreeDirty": bool(status),
        "workingTreeStatusSha256": sha256(status.encode()).hexdigest(),
        "candidateEligible": not bool(status),
        "binarySha256": digest,
        "contractVersions": [
            "mimir.metadata/v1",
            "mimir.metadata-validation/v1",
            "mimir.schedule-request/v1",
            "mimir.schedule-request-ack/v1",
            "mimir.failure/v1",
        ],
    }
    manifest_path = DIST / "manifest.json"
    manifest_path.write_text(json.dumps(manifest, sort_keys=True, indent=2) + "\n", encoding="utf-8")
    archive_path = DIST / "mimir-local.tar.gz"
    with archive_path.open("wb") as raw_archive:
        with gzip.GzipFile(fileobj=raw_archive, mode="wb", mtime=0) as compressed:
            with tarfile.open(fileobj=compressed, mode="w") as archive:
                for path, name, mode in ((BIN, "mimir", 0o755), (manifest_path, "manifest.json", 0o644)):
                    info = tarfile.TarInfo(name)
                    info.size = path.stat().st_size
                    info.mode = mode
                    info.uid = info.gid = 0
                    info.uname = info.gname = ""
                    info.mtime = 0
                    with path.open("rb") as source:
                        archive.addfile(info, source)
    print(f"created {archive_path.relative_to(ROOT)} sha256={sha256(archive_path.read_bytes()).hexdigest()}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
