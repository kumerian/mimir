#!/usr/bin/env python3
"""Verify required-check failures are visible and propagated to CI."""

import subprocess
import sys
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
RUNNER = ROOT / "scripts/required_check.py"


class RequiredCheckTests(unittest.TestCase):
    def run_check(self, name: str, exit_code: int) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [
                sys.executable,
                str(RUNNER),
                name,
                "--",
                sys.executable,
                "-c",
                f"raise SystemExit({exit_code})",
            ],
            cwd=ROOT,
            text=True,
            capture_output=True,
            check=False,
        )

    def test_required_failure_is_named_and_nonzero(self) -> None:
        result = self.run_check("deliberately-failing-required-check", 23)
        self.assertEqual(result.returncode, 23)
        self.assertIn("FAIL: required check 'deliberately-failing-required-check'", result.stderr)
        self.assertNotIn("NOT APPLICABLE", result.stdout + result.stderr)

    def test_required_success_is_explicit(self) -> None:
        result = self.run_check("synthetic-success", 0)
        self.assertEqual(result.returncode, 0)
        self.assertIn("PASS: required check 'synthetic-success'", result.stdout)


if __name__ == "__main__":
    unittest.main()
