#!/usr/bin/env python3
"""Build into a clean temporary install and exercise the local CLI end to end."""

import json
import os
from pathlib import Path
import platform
import subprocess
import sys
import tempfile


ROOT = Path(__file__).resolve().parents[1]


def run(command: list[str], cwd: Path) -> subprocess.CompletedProcess[str]:
    result = subprocess.run(command, cwd=cwd, text=True, capture_output=True, check=False)
    if result.returncode:
        print(f"FAIL: {' '.join(command)} returned {result.returncode}", file=sys.stderr)
        if result.stdout:
            print(result.stdout, file=sys.stderr, end="")
        if result.stderr:
            print(result.stderr, file=sys.stderr, end="")
        raise SystemExit(result.returncode)
    return result


def main() -> int:
    host = platform.system().lower()
    executable_name = "mimir.exe" if host == "windows" else "mimir"
    with tempfile.TemporaryDirectory(prefix="mimir-clean-install-") as directory:
        install = Path(directory) / "install"
        install.mkdir(mode=0o700)
        binary = install / executable_name
        build_env = os.environ.copy()
        build_env.pop("GOOS", None)
        build_env.pop("GOARCH", None)
        result = subprocess.run(
            ["go", "build", "-trimpath", "-o", str(binary), "./cmd/mimir"],
            cwd=ROOT, env=build_env, text=True, capture_output=True, check=False,
        )
        if result.returncode:
            print("FAIL: clean install build failed", file=sys.stderr)
            print(result.stderr, file=sys.stderr, end="")
            return result.returncode

        executable = str(binary)
        health = json.loads(run([executable, "health"], install).stdout)
        contracts = set(health.get("contracts", []))
        if health.get("status") != "ready" or not {
            "mimir.metadata/v1",
            "mimir.metadata-validation/v1",
            "mimir.schedule-request-ack/v1",
        }.issubset(contracts):
            raise SystemExit("FAIL: installed binary health/contract smoke failed")

        metadata_path = ROOT / "contracts/examples/metadata-v1-draft.json"
        state_dir = install / "state"
        report_output = run([
            executable, "validate", "--metadata", str(metadata_path),
            "--state-dir", str(state_dir),
        ], install).stdout
        report = json.loads(report_output)
        if not report.get("valid") or report.get("runEligible") or not report.get("inputDigest"):
            raise SystemExit("FAIL: installed binary metadata validation smoke failed")
        queried = json.loads(run([
            executable, "validation-report", "--state-dir", str(state_dir),
            "--input-digest", report["inputDigest"],
        ], install).stdout)
        if queried != report:
            raise SystemExit("FAIL: installed binary validation report query differs")

        source = install / "source"
        source.mkdir(mode=0o700)
        (source / "orders.jsonl").write_text('{"order_id":1,"customer_name":"Synthetic"}\n', encoding="utf-8")
        run([
            executable, "activate", "--metadata", str(metadata_path),
            "--state-dir", str(state_dir), "--allow-local",
        ], install)
        run([
            executable, "run", "--source-root", str(source),
            "--state-dir", str(state_dir), "--allow-local",
        ], install)
        print(f"PASS: clean install, health, validation, report query, and synthetic run ({platform.system()}/{platform.machine()})")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
