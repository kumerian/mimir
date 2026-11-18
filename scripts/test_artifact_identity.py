#!/usr/bin/env python3
"""Verify candidate identity is derived from the exact executable and contracts."""

import hashlib
import json
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch


sys.path.insert(0, str(Path(__file__).resolve().parent))
import artifact_identity


class ArtifactIdentityTests(unittest.TestCase):
    def test_candidate_identity_binds_binary_revision_and_runtime_contracts(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            binary = Path(directory) / "mimir"
            binary.write_bytes(b"synthetic executable")
            health = json.dumps({
                "status": "ready",
                "contracts": ["mimir.failure/v1", "mimir.metadata/v1"],
            })
            with patch.object(artifact_identity.subprocess, "run") as run:
                run.return_value.returncode = 0
                run.return_value.stdout = health
                run.return_value.stderr = ""
                identity = artifact_identity.create_identity(
                    binary=binary,
                    source_revision="a" * 40,
                    working_tree_status="",
                )

        self.assertEqual(identity["sourceRevision"], "a" * 40)
        self.assertEqual(identity["binarySha256"], hashlib.sha256(b"synthetic executable").hexdigest())
        self.assertEqual(identity["contractVersions"], ["mimir.failure/v1", "mimir.metadata/v1"])
        self.assertTrue(identity["candidateEligible"])
        self.assertEqual(identity["workingTreeStatusSha256"], hashlib.sha256(b"").hexdigest())
        self.assertEqual(run.call_args.args[0][-1], "health")

    def test_dirty_candidate_is_marked_ineligible_and_status_is_fingerprinted(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            binary = Path(directory) / "mimir"
            binary.write_bytes(b"synthetic executable")
            with patch.object(artifact_identity.subprocess, "run") as run:
                run.return_value.returncode = 0
                run.return_value.stdout = '{"status":"ready","contracts":["mimir.metadata/v1"]}'
                run.return_value.stderr = ""
                identity = artifact_identity.create_identity(
                    binary=binary,
                    source_revision="b" * 40,
                    working_tree_status=" M README.md\n",
                )

        self.assertFalse(identity["candidateEligible"])
        self.assertEqual(
            identity["workingTreeStatusSha256"],
            hashlib.sha256(b" M README.md\n").hexdigest(),
        )

    def test_identity_rejects_unready_or_incomplete_health_output(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            binary = Path(directory) / "mimir"
            binary.write_bytes(b"synthetic executable")
            for health in (
                '{"status":"not_ready","contracts":["mimir.metadata/v1"]}',
                '{"status":"ready","contracts":[]}',
                '{"status":"ready","contracts":[""]}',
            ):
                with self.subTest(health=health):
                    with patch.object(artifact_identity.subprocess, "run") as run:
                        run.return_value.returncode = 0
                        run.return_value.stdout = health
                        run.return_value.stderr = ""
                        with self.assertRaises(ValueError):
                            artifact_identity.create_identity(
                                binary=binary,
                                source_revision="c" * 40,
                                working_tree_status="",
                            )

    def test_identity_rejects_failed_health_command(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            binary = Path(directory) / "mimir"
            binary.write_bytes(b"synthetic executable")
            with patch.object(artifact_identity.subprocess, "run") as run:
                run.return_value.returncode = 1
                run.return_value.stdout = ""
                run.return_value.stderr = "synthetic failure"
                with self.assertRaises(RuntimeError):
                    artifact_identity.create_identity(
                        binary=binary,
                        source_revision="d" * 40,
                        working_tree_status="",
                    )


if __name__ == "__main__":
    unittest.main()
