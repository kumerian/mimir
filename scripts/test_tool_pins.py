#!/usr/bin/env python3
"""Keep local and CI vulnerability scanners on the same pinned release."""

import re
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


class ToolPinTests(unittest.TestCase):
    def test_local_and_ci_govulncheck_versions_match(self) -> None:
        makefile = (ROOT / "Makefile").read_text(encoding="utf-8")
        workflow = (ROOT / ".github/workflows/ci.yml").read_text(encoding="utf-8")
        pattern = r"golang\.org/x/vuln/cmd/govulncheck@(v[0-9.]+)"
        local_versions = re.findall(pattern, makefile)
        ci_versions = re.findall(pattern, workflow)

        self.assertEqual(local_versions, ["v1.10.0"])
        self.assertEqual(ci_versions, ["v1.10.0"])


if __name__ == "__main__":
    unittest.main()
