#!/usr/bin/env python3
"""Unit tests for architecture dependency enforcement."""

import tempfile
import unittest
from pathlib import Path

from check_architecture import dependency_violations


class ArchitectureCheckTests(unittest.TestCase):
    def test_invalid_inward_dependency_is_identified(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            package = root / "internal/domain"
            package.mkdir(parents=True)
            (package / "bad.go").write_text(
                'package domain\nimport "github.com/kumerian/mimir/internal/local"\n',
                encoding="utf-8",
            )
            violations = dependency_violations(
                root, {"internal/domain": ("internal/local",)}
            )
            self.assertEqual(len(violations), 1)
            self.assertIn("internal/local", violations[0])

    def test_inward_dependency_is_allowed(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            package = root / "internal/domain"
            package.mkdir(parents=True)
            (package / "good.go").write_text(
                'package domain\nimport "fmt"\n',
                encoding="utf-8",
            )
            self.assertEqual(
                dependency_violations(root, {"internal/domain": ("internal/local",)}),
                [],
            )


if __name__ == "__main__":
    unittest.main()
