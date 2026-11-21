#!/usr/bin/env python3
import json
from pathlib import Path
import subprocess
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[1]


class GuiCandidateIdentityTests(unittest.TestCase):
    def test_identity_binds_binary_frontend_and_contracts(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            binary = root / "mimir"
            binary.write_text(
                "#!/bin/sh\nprintf '%s\\n' '{\"status\":\"ready\",\"contracts\":[\"mimir.gui-api/v1\"]}'\n",
                encoding="utf-8",
            )
            binary.chmod(0o755)
            dist = root / "dist"
            dist.mkdir()
            (dist / "index.html").write_text("<html></html>\n", encoding="utf-8")
            (dist / "app.js").write_text("console.log('synthetic');\n", encoding="utf-8")
            output = root / "identity.json"

            result = subprocess.run(
                [
                    "python3",
                    str(ROOT / "scripts/gui_candidate_identity.py"),
                    "--binary",
                    str(binary),
                    "--frontend-dist",
                    str(dist),
                    "--output",
                    str(output),
                ],
                cwd=ROOT,
                check=False,
                text=True,
                capture_output=True,
            )
            self.assertEqual(result.returncode, 0, result.stderr)
            identity = json.loads(output.read_text(encoding="utf-8"))
            self.assertEqual(identity["contractVersions"], ["mimir.gui-api/v1"])
            self.assertEqual(identity["frontend"]["fileCount"], 2)
            self.assertTrue(identity["binarySha256"])
            self.assertTrue(identity["frontend"]["sha256"])


if __name__ == "__main__":
    unittest.main()
