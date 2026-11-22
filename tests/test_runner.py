"""Tests for the cross-platform test-suite orchestration."""

import os
from pathlib import Path
import subprocess
from types import SimpleNamespace
import unittest
from unittest.mock import patch

from scripts import run_tests


class TestRunnerTest(unittest.TestCase):
    def setUp(self):
        which = patch("scripts.run_tests.shutil.which", return_value="npm")
        which.start()
        self.addCleanup(which.stop)

    @patch("scripts.run_tests.subprocess.run")
    def test_runs_suites_with_isolated_stack_and_dynamic_port(self, run):
        run.return_value = SimpleNamespace(stdout="127.0.0.1:49152\n")
        run_tests.run_tests(["--no-build"])

        calls = run.call_args_list
        self.assertEqual(len(calls), 7)
        self.assertEqual(
            Path(calls[0].args[0][0]).name,
            "mvnw.cmd" if os.name == "nt" else "mvnw",
        )
        self.assertEqual(calls[1].args[0], ["npm.cmd" if os.name == "nt" else "npm", "ci"])
        self.assertEqual(calls[2].args[0][-1], "test")
        self.assertIn("--no-build", calls[3].args[0])
        env = calls[3].kwargs["env"]
        self.assertTrue(env["COMPOSE_PROJECT_NAME"].startswith("mimir-test-"))
        self.assertEqual(env["FRONTEND_PORT"], "0")
        self.assertTrue(env["DB_PASSWORD"])
        self.assertTrue(env["DB_ROOT_PASSWORD"])
        self.assertNotEqual(env["DB_PASSWORD"], env["DB_ROOT_PASSWORD"])
        self.assertEqual(calls[6].kwargs["env"]["COMPOSE_PROJECT_NAME"], env["COMPOSE_PROJECT_NAME"])
        self.assertEqual(calls[6].args[0][-3:], ["down", "--volumes", "--remove-orphans"])
        self.assertEqual(calls[4].args[0][-3:], ["port", "frontend", "80"])
        self.assertEqual(calls[5].args[0][1:4], ["-m", "unittest", "discover"])
        self.assertEqual(env["STACK_URL"], "http://127.0.0.1:49152")

    @patch("scripts.run_tests.shutil.which", return_value=None)
    @patch("scripts.run_tests.subprocess.run")
    def test_reports_missing_npm_before_running_suites(self, run, _which):
        with self.assertRaisesRegex(FileNotFoundError, "install Node.js 24 with npm"):
            run_tests.run_tests(["--no-build"])
        run.assert_not_called()

    @patch("scripts.run_tests.shutil.which", side_effect=[None, "fnm"])
    @patch("scripts.run_tests.subprocess.run")
    def test_uses_fnm_when_npm_is_not_on_path(self, run, _which):
        run.return_value = SimpleNamespace(stdout="127.0.0.1:49152\n")
        run_tests.run_tests(["--no-build"])
        npm = "npm.cmd" if os.name == "nt" else "npm"
        command = ["fnm", "exec", "--using", "24", "--", npm]
        self.assertEqual(run.call_args_list[1].args[0], [*command, "ci"])
        self.assertEqual(run.call_args_list[2].args[0], [*command, "test"])

    @patch.dict(os.environ, {"DB_PASSWORD": "existing-db-value", "DB_ROOT_PASSWORD": "existing-root-value"})
    @patch("scripts.run_tests.subprocess.run")
    def test_test_credentials_do_not_use_or_change_local_stack_credentials(self, run):
        run.return_value = SimpleNamespace(stdout="127.0.0.1:49152\n")
        run_tests.run_tests(["--no-build"])
        env = run.call_args_list[3].kwargs["env"]
        self.assertNotEqual(env["DB_PASSWORD"], os.environ["DB_PASSWORD"])
        self.assertNotEqual(env["DB_ROOT_PASSWORD"], os.environ["DB_ROOT_PASSWORD"])
        self.assertEqual(os.environ["DB_PASSWORD"], "existing-db-value")
        self.assertEqual(os.environ["DB_ROOT_PASSWORD"], "existing-root-value")

    @patch("scripts.run_tests.subprocess.run")
    def test_cleans_up_if_compose_start_fails(self, run):
        run.side_effect = [None, None, None, subprocess.CalledProcessError(1, "docker"), None, None]
        with self.assertRaises(subprocess.CalledProcessError):
            run_tests.run_tests(["--build"])
        self.assertEqual(run.call_args_list[-2].args[0][-2:], ["logs", "--no-color"])
        self.assertEqual(run.call_args.args[0][-3:], ["down", "--volumes", "--remove-orphans"])

    @patch("scripts.run_tests.subprocess.run")
    def test_cleans_up_if_integration_tests_fail(self, run):
        run.side_effect = [
            None, None, None, None, SimpleNamespace(stdout="127.0.0.1:49152\n"),
            subprocess.CalledProcessError(1, "unittest"), None, None,
        ]
        with self.assertRaises(subprocess.CalledProcessError):
            run_tests.run_tests(["--build"])
        self.assertEqual(run.call_args.args[0][-3:], ["down", "--volumes", "--remove-orphans"])

    @patch("scripts.run_tests.subprocess.run")
    def test_does_not_start_or_clean_stack_if_application_test_fails(self, run):
        run.side_effect = [None, subprocess.CalledProcessError(1, "npm")]
        with self.assertRaises(subprocess.CalledProcessError):
            run_tests.run_tests(["--build"])
        self.assertEqual(run.call_count, 2)


if __name__ == "__main__":
    unittest.main()
