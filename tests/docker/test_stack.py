"""Integration checks against an already-running, healthy Compose stack."""

import json
import os
import re
import subprocess
import unittest
import uuid
from urllib.error import HTTPError
from urllib.parse import urlencode
from urllib.request import Request, urlopen


BASE_URL = os.environ.get("STACK_URL", "http://127.0.0.1:8080").rstrip("/")


def fetch(path):
    with urlopen(f"{BASE_URL}{path}", timeout=10) as response:
        return response.headers, response.read().decode("utf-8")


class DockerStackTest(unittest.TestCase):
    def test_metadata_crud_relationships_and_persistence_after_restart(self):
        suffix = uuid.uuid4().hex
        source = {
            "sourceName": f"source/table & {suffix}",
            "sourceFunctionalName": "Integration source",
            "sourceType": "database",
            "sourceOwner": "Integration owner",
            "sourceApplication": "CRM",
            "sourceLocation": "https://integration",
            "isActive": True,
        }
        authentication = {
            "authenticationId": f"auth-{suffix}",
            "authenticationType": "vault",
            "authenticationLocation": "vault://integration/reference",
            "authenticationUser": "integration-user",
            "authenticationCredential": "local-test-credential",
            "isActive": True,
        }
        interface = {
            "interfaceName": f"interface-{suffix}",
            "sourceName": source["sourceName"],
            "authenticationId": authentication["authenticationId"],
            "interfaceFunctionalName": "Integration interface",
            "interfaceType": "jdbc",
            "interfaceLocation": "jdbc:mariadb://source",
            "decimalSeparator": ".",
            "thousandsSeparator": ",",
            "delimiter": ";",
            "hasHeaders": True,
            "isActive": True,
        }
        source_object = {
            "objectName": f"object-{suffix}",
            "interfaceName": interface["interfaceName"],
            "objectFunctionalName": "Integration object",
            "objectDescription": "Integration description",
            "isConfidential": False,
            "isBusinessCritical": True,
            "isActive": True,
        }
        api_parameter = {
            "parameterId": f"parameter-{suffix}",
            "interfaceId": interface["interfaceName"],
            "parameterName": "api-version",
            "parameterValue": "v1",
        }
        column = {
            "columnName": f"column-{suffix}",
            "objectName": source_object["objectName"],
            "columnAlternateName": "integration_column",
            "columnFunctionalName": "Integration column",
            "columnDescription": "Integration column description",
            "columnOrdinalPosition": 1,
            "needsEncryption": False,
            "isPartitionField": False,
            "partitionOrdinalPosition": 0,
            "isKey": True,
            "keyOrdinalPosition": 1,
            "isNullable": False,
            "testDatatype": True,
            "datatype": "varchar(255)",
            "characterMaximumLength": 255,
            "numericScale": None,
            "numericPrecision": None,
            "nullValueRegexPattern": None,
            "isActive": True,
        }
        entities = [
            ("source-systems", "sourceName", source),
            ("authentications", "authenticationId", authentication),
            ("interfaces", "interfaceName", interface),
            ("api-parameters", "parameterId", api_parameter),
            ("objects", "objectName", source_object),
            ("columns", "columnName", column),
        ]

        def request(method, resource, value=None, identifier=None):
            path = f"/api/metadata/{resource}"
            if identifier is not None:
                path += "?" + urlencode({"id": identifier})
            data = None if value is None else json.dumps(value).encode("utf-8")
            req = Request(
                f"{BASE_URL}{path}", data=data, method=method,
                headers={"Content-Type": "application/json"},
            )
            with urlopen(req, timeout=20) as response:
                body = response.read()
                return response.status, json.loads(body) if body else None

        created = []
        try:
            for resource, key, value in entities:
                status, body = request("POST", resource, value)
                created.append((resource, key, value))
                self.assertEqual(status, 201)
                self.assertEqual(body, value)
                self.assertEqual(request("GET", resource, identifier=value[key])[1], value)
                self.assertIn(value, request("GET", resource)[1])
                with self.assertRaises(HTTPError) as duplicate:
                    request("POST", resource, value)
                self.assertEqual(duplicate.exception.code, 409)

            for resource, key, value in entities:
                if resource in ("api-parameters", "columns"):
                    continue
                with self.assertRaises(HTTPError) as referenced:
                    request("DELETE", resource, identifier=value[key])
                self.assertEqual(referenced.exception.code, 409)

            subprocess.run(
                ["docker", "compose", "restart", "backend"], check=True,
            )
            subprocess.run(
                ["docker", "compose", "up", "--no-build", "--no-recreate", "--detach", "--wait",
                 "--wait-timeout", "180", "backend"],
                check=True,
            )
            for resource, key, value in entities:
                self.assertEqual(request("GET", resource, identifier=value[key])[1], value)
                updated = dict(value)
                functional = (
                    "authenticationType" if resource == "authentications"
                    else "parameterName" if resource == "api-parameters"
                    else next(field for field in value if field.endswith("FunctionalName"))
                )
                updated[functional] = "Updated"
                self.assertEqual(request("PUT", resource, updated, value[key]), (200, updated))
                self.assertEqual(request("GET", resource, identifier=value[key])[1], updated)
        finally:
            for resource, key, value in reversed(created):
                self.assertEqual(request("DELETE", resource, identifier=value[key])[0], 204)
                with self.assertRaises(HTTPError) as missing:
                    request("GET", resource, identifier=value[key])
                self.assertEqual(missing.exception.code, 404)

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
