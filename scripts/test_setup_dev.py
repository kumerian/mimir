#!/usr/bin/env python3
"""Tests for the local development toolchain installer."""

import hashlib
import io
import json
import tarfile
import tempfile
import unittest
import zipfile
from pathlib import Path
from unittest.mock import patch

import setup_dev


class SetupDevTests(unittest.TestCase):
    def test_resolves_installed_pinned_go_on_windows(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            home = Path(temp)
            go = home / ".mimir-dev" / "go1.27.3" / "bin" / "go.exe"
            go.parent.mkdir(parents=True)
            go.write_bytes(b"synthetic go")
            self.assertEqual(setup_dev.resolve_go_command(home, "1.27.3", "windows"), str(go))

    def test_resolves_installed_pinned_gofmt_on_windows(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            home = Path(temp)
            gofmt = home / ".mimir-dev" / "go1.27.3" / "bin" / "gofmt.exe"
            gofmt.parent.mkdir(parents=True)
            gofmt.write_bytes(b"synthetic gofmt")
            self.assertEqual(setup_dev.resolve_tool_command(home, "1.27.3", "windows", "gofmt"), str(gofmt))

    def test_missing_pinned_go_falls_back_to_path(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            self.assertEqual(setup_dev.resolve_go_command(Path(temp), "1.27.3", "windows"), "go")

    def test_installs_verified_go_archive_and_reuses_it(self) -> None:
        archive = io.BytesIO()
        with tarfile.open(fileobj=archive, mode="w:gz") as tar:
            data = b"synthetic go"
            entry = tarfile.TarInfo("go/bin/go")
            entry.size = len(data)
            tar.addfile(entry, io.BytesIO(data))
        payload = archive.getvalue()
        release = [{"version": "go1.27.3", "files": [{
            "filename": "go1.27.3.linux-amd64.tar.gz",
            "sha256": hashlib.sha256(payload).hexdigest(),
        }]}]
        with tempfile.TemporaryDirectory() as temp:
            with patch.object(setup_dev, "download", side_effect=[json.dumps(release).encode(), payload]):
                path = setup_dev.install_go(Path(temp), "1.27.3", "linux", "amd64")
            self.assertEqual((path / "bin/go").read_bytes(), b"synthetic go")
            with patch.object(setup_dev, "download", side_effect=AssertionError("unexpected download")):
                self.assertEqual(setup_dev.install_go(Path(temp), "1.27.3", "linux", "amd64"), path)

    def test_checksum_failure_does_not_install_go(self) -> None:
        release = [{"version": "go1.27.3", "files": [{
            "filename": "go1.27.3.linux-amd64.tar.gz", "sha256": "0" * 64,
        }]}]
        with tempfile.TemporaryDirectory() as temp:
            with patch.object(setup_dev, "download", side_effect=[json.dumps(release).encode(), b"bad"]):
                with self.assertRaisesRegex(ValueError, "checksum"):
                    setup_dev.install_go(Path(temp), "1.27.3", "linux", "amd64")
            self.assertFalse((Path(temp) / "go1.27.3").exists())

    def test_missing_release_is_explicit(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            with patch.object(setup_dev, "download", return_value=b"[]"):
                with self.assertRaisesRegex(ValueError, "not found"):
                    setup_dev.install_go(Path(temp), "1.27.3", "linux", "arm64")

    def test_windows_zip_is_installed(self) -> None:
        archive = io.BytesIO()
        with zipfile.ZipFile(archive, "w") as bundle:
            bundle.writestr("go/bin/go.exe", b"synthetic go")
        payload = archive.getvalue()
        release = [{"version": "go1.27.3", "files": [{
            "filename": "go1.27.3.windows-arm64.zip",
            "sha256": hashlib.sha256(payload).hexdigest(),
        }]}]
        with tempfile.TemporaryDirectory() as temp:
            with patch.object(setup_dev, "download", side_effect=[json.dumps(release).encode(), payload]):
                path = setup_dev.install_go(Path(temp), "1.27.3", "windows", "arm64")
            self.assertEqual((path / "bin/go.exe").read_bytes(), b"synthetic go")

    def test_incomplete_install_is_not_overwritten(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            existing = Path(temp) / "go1.27.3"
            existing.mkdir()
            with patch.object(setup_dev, "download", side_effect=AssertionError("unexpected download")):
                with self.assertRaisesRegex(ValueError, "incomplete installation"):
                    setup_dev.install_go(Path(temp), "1.27.3", "linux", "amd64")
            self.assertTrue(existing.is_dir())


if __name__ == "__main__":
    unittest.main()
