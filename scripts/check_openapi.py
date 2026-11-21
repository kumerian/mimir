#!/usr/bin/env python3
"""Check the local-profile OpenAPI contract using only the Python standard library."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

EXPECTED_PATHS = {
    "/health": {"get"},
    "/metadata/validate": {"post"},
    "/metadata/validation-reports/{inputDigest}": {"get"},
    "/metadata/revisions": {"get", "post"},
    "/metadata/activate": {"post"},
    "/discovery": {"post"},
    "/runs": {"post"},
    "/schedules/evaluate": {"post"},
    "/schedules/stop": {"post"},
    "/replays": {"post"},
    "/cleaning/preview": {"post"},
    "/cleaning/apply": {"post"},
}
CONTRACT_VERSION = "mimir.gui-api/v1"
METHODS = {"get", "put", "post", "delete", "options", "head", "patch", "trace"}


def fail(message: str) -> None:
    raise ValueError(message)


def resolve_reference(reference: str, contract_path: Path, document: dict) -> None:
    if reference.startswith("#/"):
        value: object = document
        for token in reference[2:].split("/"):
            token = token.replace("~1", "/").replace("~0", "~")
            if not isinstance(value, dict) or token not in value:
                fail(f"unresolved local reference: {reference}")
            value = value[token]
        return
    target_text, separator, fragment = reference.partition("#")
    target = (contract_path.parent / target_text).resolve()
    if not target.is_file():
        fail(f"unresolved external reference: {reference}")
    try:
        external = json.loads(target.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        fail(f"invalid external JSON reference {reference}: {error}")
    if separator and fragment:
        value = external
        for token in fragment.lstrip("/").split("/"):
            token = token.replace("~1", "/").replace("~0", "~")
            if not isinstance(value, dict) or token not in value:
                fail(f"unresolved external reference: {reference}")
            value = value[token]


def walk_references(value: object, contract_path: Path, document: dict) -> None:
    if isinstance(value, dict):
        reference = value.get("$ref")
        if isinstance(reference, str):
            resolve_reference(reference, contract_path, document)
        for child in value.values():
            walk_references(child, contract_path, document)
    elif isinstance(value, list):
        for child in value:
            walk_references(child, contract_path, document)


def check(contract_path: Path) -> None:
    try:
        document = json.loads(contract_path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        fail(f"cannot read OpenAPI JSON: {error}")
    if document.get("openapi") != "3.1.0":
        fail("contract must use OpenAPI 3.1.0")
    if document.get("info", {}).get("version") != CONTRACT_VERSION:
        fail("contract info.version does not match the supported API version")
    servers = document.get("servers", [])
    if not servers or servers[0].get("url") != "http://127.0.0.1:8787/api/v1":
        fail("contract must document the loopback /api/v1 default server")
    paths = document.get("paths")
    if not isinstance(paths, dict) or set(paths) != set(EXPECTED_PATHS):
        fail("contract paths do not match the approved local-profile operation set")

    operation_ids: set[str] = set()
    for route, expected_methods in EXPECTED_PATHS.items():
        path_item = paths[route]
        actual_methods = {name for name in path_item if name.lower() in METHODS}
        if actual_methods != expected_methods:
            fail(f"{route} methods are {sorted(actual_methods)}, expected {sorted(expected_methods)}")
        for method in expected_methods:
            operation = path_item[method]
            operation_id = operation.get("operationId")
            if not operation_id or operation_id in operation_ids:
                fail(f"missing or duplicate operationId for {method.upper()} {route}")
            operation_ids.add(operation_id)
            parameters = operation.get("parameters", [])
            version = [
                parameter for parameter in parameters
                if parameter.get("$ref") == "#/components/parameters/ContractVersion"
                or (
                    parameter.get("name") == "Mimir-Contract-Version"
                    and parameter.get("in") == "header"
                    and parameter.get("required") is True
                    and parameter.get("schema", {}).get("const") == CONTRACT_VERSION
                )
            ]
            if not version:
                fail(f"{method.upper()} {route} lacks the required contract-version header")
            if method == "post":
                if not any(
                    parameter.get("$ref") == "#/components/parameters/SameOrigin"
                    for parameter in parameters
                ):
                    fail(f"POST {route} lacks the required same-origin parameter")
                content = operation.get("requestBody", {}).get("content", {})
                if "application/json" not in content:
                    fail(f"{method.upper()} {route} lacks an application/json request body")
            default = operation.get("responses", {}).get("default", {})
            if default.get("$ref") != "#/components/responses/Failure":
                fail(f"{method.upper()} {route} lacks the stable failure response")
    parameter_components = document.get("components", {}).get("parameters", {})
    contract_version = parameter_components.get("ContractVersion", {})
    if (
        contract_version.get("name") != "Mimir-Contract-Version"
        or contract_version.get("in") != "header"
        or contract_version.get("required") is not True
        or contract_version.get("schema", {}).get("const") != CONTRACT_VERSION
    ):
        fail("ContractVersion parameter must require the supported Mimir-Contract-Version header")
    same_origin = parameter_components.get("SameOrigin", {})
    if (
        same_origin.get("name") != "Origin"
        or same_origin.get("in") != "header"
        or same_origin.get("required") is not True
    ):
        fail("SameOrigin parameter must require the browser Origin header")
    walk_references(document, contract_path, document)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--contract",
        type=Path,
        default=Path("contracts/openapi/local-profile-v1.json"),
    )
    arguments = parser.parse_args()
    try:
        check(arguments.contract)
    except ValueError as error:
        print(f"OpenAPI check failed: {error}", file=sys.stderr)
        return 1
    print("OpenAPI 3.1 contract: passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
