#!/usr/bin/env python3
"""Tests for race-test prerequisite diagnostics."""

import contextlib
import io
import unittest
from unittest.mock import patch

import run_race_tests


class RunRaceTests(unittest.TestCase):
    def test_missing_compiler_has_actionable_error(self) -> None:
        stderr = io.StringIO()
        with (
            patch.object(run_race_tests, "find_compiler", return_value=None),
            contextlib.redirect_stderr(stderr),
        ):
            result = run_race_tests.run("go")
        self.assertEqual(result, 2)
        self.assertIn("race tests require a C compiler", stderr.getvalue())
        self.assertIn("docs/operations/local-development.md", stderr.getvalue())

    def test_race_tests_enable_cgo_and_run(self) -> None:
        completed = unittest.mock.Mock(returncode=0)
        with (
            patch.object(run_race_tests, "find_compiler", return_value="gcc"),
            patch.object(run_race_tests.subprocess, "run", return_value=completed) as subprocess_run,
        ):
            result = run_race_tests.run("go")
        self.assertEqual(result, 0)
        command = subprocess_run.call_args.args[0]
        self.assertEqual(command, ["go", "test", "-race", "./..."])
        self.assertEqual(subprocess_run.call_args.kwargs["env"]["CGO_ENABLED"], "1")


if __name__ == "__main__":
    unittest.main()
