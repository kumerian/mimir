"""Integration checks against an already-running, healthy Compose stack."""

import json
import os
import re
import subprocess
import unittest
from urllib.error import HTTPError
from urllib.request import urlopen


BASE_URL = os.environ.get("STACK_URL", "http://127.0.0.1:8080").rstrip("/")


def fetch(path):
    with urlopen(f"{BASE_URL}{path}", timeout=10) as response:
        return response.headers, response.read().decode("utf-8")


class DockerStackTest(unittest.TestCase):
    def test_exactly_three_healthy_containers(self):
        result = subprocess.run(
            ["docker", "compose", "ps", "--format", "json"],
            check=True, capture_output=True, text=True,
        )
        containers = [json.loads(line) for line in result.stdout.splitlines() if line]
        self.assertEqual({item["Service"] for item in containers},
                         {"backend", "frontend", "database"})
        self.assertEqual(len(containers), 3)
        for item in containers:
            self.assertEqual(item["State"], "running", item["Service"])
            self.assertEqual(item["Health"], "healthy", item["Service"])

    def test_frontend_serves_angular_and_its_bundle(self):
        _, html = fetch("/")
        self.assertIn("<app-root>", html)
        script = re.search(r'<script[^>]+src="([^"]+\.js)"', html)
        self.assertIsNotNone(script, "Angular JavaScript bundle was not linked")
        headers, bundle = fetch("/" + script.group(1).lstrip("/"))
        self.assertIn("javascript", headers.get("Content-Type"))
        self.assertTrue(bundle.strip())

    def test_frontend_proxies_to_backend_with_real_database(self):
        headers, body = fetch("/api/status")
        self.assertIn("application/json", headers.get("Content-Type"))
        self.assertEqual(json.loads(body), {
            "application": "mimir", "database": "connected",
        })

    def test_spa_navigation_falls_back_to_index(self):
        _, index = fetch("/")
        _, nested = fetch("/ingestion/example")
        self.assertEqual(nested, index)

    def test_missing_assets_and_api_routes_are_not_spa_successes(self):
        for path in ("/missing.js", "/api/missing"):
            with self.subTest(path=path):
                with self.assertRaises(HTTPError) as error:
                    fetch(path)
                self.assertEqual(error.exception.code, 404)


if __name__ == "__main__":
    unittest.main()
