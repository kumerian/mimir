#!/usr/bin/env python3
"""Install the repository-pinned Go toolchain in the user's home directory."""

import hashlib
import io
import json
import os
import platform
import re
import subprocess
import sys
import tarfile
import tempfile
import urllib.request
import zipfile
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
RELEASES = "https://go.dev/dl/?mode=json&include=all"


def download(url: str) -> bytes:
    with urllib.request.urlopen(url, timeout=120) as response:
        return response.read()


def install_go(home: Path, version: str, os_name: str, arch: str) -> Path:
    destination = home / f"go{version}"
    executable = "go.exe" if os_name == "windows" else "go"
    if destination.exists():
        if not (destination / "bin" / executable).is_file():
            raise ValueError(f"incomplete installation at {destination}; remove it before retrying")
        return destination

    extension = "zip" if os_name == "windows" else "tar.gz"
    filename = f"go{version}.{os_name}-{arch}.{extension}"
    releases = json.loads(download(RELEASES))
    matches = [
        item for release in releases if release["version"] == f"go{version}"
        for item in release["files"] if item["filename"] == filename
    ]
    if len(matches) != 1:
        raise ValueError(f"official Go release {filename} not found")
    archive = download(f"https://go.dev/dl/{filename}")
    if hashlib.sha256(archive).hexdigest() != matches[0]["sha256"]:
        raise ValueError(f"checksum mismatch for {filename}")

    home.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix=".install-", dir=home) as temp:
        staging = Path(temp)
        if extension == "zip":
            with zipfile.ZipFile(io.BytesIO(archive)) as bundle:
                bundle.extractall(staging)
        else:
            with tarfile.open(fileobj=io.BytesIO(archive), mode="r:gz") as bundle:
                bundle.extractall(staging, filter="data")
        if not (staging / "go" / "bin" / executable).is_file():
            raise ValueError(f"archive {filename} does not contain the Go executable")
        (staging / "go").rename(destination)
    return destination


def main() -> int:
    if sys.version_info[:2] != (3, 12):
        raise ValueError("Python 3.12 is required")
    match = re.search(r"(?m)^toolchain go(\d+\.\d+\.\d+)\s*$", (ROOT / "go.mod").read_text())
    if not match:
        raise ValueError("go.mod does not contain a pinned Go toolchain")
    os_name = {"Linux": "linux", "Darwin": "darwin", "Windows": "windows"}.get(platform.system())
    arch = {"x86_64": "amd64", "AMD64": "amd64", "aarch64": "arm64", "arm64": "arm64", "ARM64": "arm64"}.get(platform.machine())
    if not os_name or not arch:
        raise ValueError(f"unsupported platform: {platform.system()} {platform.machine()}")
    version = match.group(1)
    go_root = install_go(Path.home() / ".mimir-dev", version, os_name, arch)
    go = go_root / "bin" / ("go.exe" if os_name == "windows" else "go")
    env = os.environ.copy()
    env["GOTOOLCHAIN"] = "local"
    result = subprocess.run([str(go), "version"], check=True, text=True, capture_output=True, env=env)
    if not result.stdout.startswith(f"go version go{version} "):
        raise ValueError(f"installed Go has an unexpected version: {result.stdout.strip()}")
    subprocess.run([str(go), "mod", "download"], cwd=ROOT, env=env, check=True)
    print(f"Installed Go {version} and downloaded module dependencies.")
    print(f"Go binary directory: {go_root / 'bin'}. See the local development setup guide for PATH configuration.")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, ValueError, subprocess.CalledProcessError) as error:
        print(f"ERROR: development setup failed: {error}", file=sys.stderr)
        raise SystemExit(1)
