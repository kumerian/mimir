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
from pathlib import Path


BASE_URL = os.environ.get("STACK_URL", "http://127.0.0.1:8080").rstrip("/")
ROOT = Path(__file__).resolve().parents[2]
FLATFILE_DIRECTORY = Path(os.environ.get("FLATFILE_DIRECTORY", ROOT / "flatfiles"))


def fetch(path):
    with urlopen(f"{BASE_URL}{path}", timeout=10) as response:
        return response.headers, response.read().decode("utf-8")


def api_request(method, path, value=None):
    data = None if value is None else json.dumps(value).encode("utf-8")
    request = Request(
        f"{BASE_URL}{path}",
        data=data,
        method=method,
        headers={"Content-Type": "application/json"},
    )
    try:
        with urlopen(request, timeout=30) as response:
            body = response.read().decode("utf-8")
            return response.status, json.loads(body) if body else None
    except HTTPError as error:
        body = error.read().decode("utf-8")
        return error.code, json.loads(body) if body else None


def database_query(query):
    result = subprocess.run(
        [
            "docker", "compose", "exec", "-T", "database", "sh", "-c",
            'mariadb --batch --skip-column-names --execute "$1" "$MARIADB_DATABASE" '
            '--user="$MARIADB_USER" --password="$MARIADB_PASSWORD"',
            "mimir-test-query", query,
        ],
        check=True, capture_output=True, text=True,
    )
    return result.stdout.strip()


def database_scalar(query):
    return int(database_query(query))


def database_execute(query):
    result = subprocess.run(
        [
            "docker", "compose", "exec", "-T", "database", "sh", "-c",
            'mariadb --batch --skip-column-names --execute "$1" "$MARIADB_DATABASE" '
            '--user="$MARIADB_USER" --password="$MARIADB_PASSWORD"',
            "mimir-test-query", query,
        ],
        capture_output=True, text=True,
    )
    if result.returncode:
        raise AssertionError(result.stderr.strip())


def database_admin_execute(query):
    result = subprocess.run(
        [
            "docker", "compose", "exec", "-T", "database", "sh", "-c",
            'MYSQL_PWD="$MARIADB_ROOT_PASSWORD" mariadb --batch --skip-column-names '
            '--execute "$1" --user=root',
            "mimir-admin-query", query,
        ],
        capture_output=True, text=True,
    )
    if result.returncode:
        raise AssertionError(result.stderr.strip())


def loaded_data_scalar(query):
    result = subprocess.run(
        [
            "docker", "compose", "exec", "-T", "database", "sh", "-c",
            'MYSQL_PWD="$LOADED_DB_PASSWORD" mariadb --batch --skip-column-names '
            '--execute "$1" --database="$LOADED_DB_NAME" --user="$LOADED_DB_USER"',
            "mimir-loaded-data-query", query,
        ],
        check=True, capture_output=True, text=True,
    )
    return result.stdout.strip()


def loaded_data_execute(query):
    result = subprocess.run(
        [
            "docker", "compose", "exec", "-T", "database", "sh", "-c",
            'MYSQL_PWD="$LOADED_DB_PASSWORD" mariadb --batch --skip-column-names '
            '--execute "$1" --database="$LOADED_DB_NAME" --user="$LOADED_DB_USER"',
            "mimir-loaded-data-query", query,
        ],
        capture_output=True, text=True,
    )
    if result.returncode:
        raise AssertionError(result.stderr.strip())
    return result.stdout.strip()


class DockerStackTest(unittest.TestCase):
    def test_flatfile_loader_derives_metadata_and_loads_csv_rows(self):
        FLATFILE_DIRECTORY.mkdir(parents=True, exist_ok=True)
        suffix = uuid.uuid4().hex
        filename = f"loader-{suffix}.csv"
        file = FLATFILE_DIRECTORY / filename
        source_name = f"loader-source-{suffix}"
        interface_id = None
        object_id = None
        table = None
        file.write_text(
            "name;amount;active\n\"Acme; Inc.\";1,234.50;true\nBeta;2.50;false\n",
            encoding="utf-8",
        )
        try:
            status, _ = api_request("POST", "/api/metadata/source-systems", {
                "sourceName": source_name,
                "sourceFunctionalName": "CSV source",
                "sourceType": "flatfile",
                "sourceOwner": "Integration",
                "sourceApplication": "Mimir test",
                "sourceLocation": "/data",
                "isActive": True,
            })
            self.assertEqual(status, 201)
            status, source_interface = api_request("POST", "/api/metadata/interfaces", {
                "interfaceName": "sample",
                "sourceName": source_name,
                "authenticationId": None,
                "interfaceFunctionalName": "CSV interface",
                "interfaceType": "csv",
                "interfaceLocation": f"/data/{filename}",
                "decimalSeparator": ".",
                "thousandsSeparator": ",",
                "delimiter": ";",
                "hasHeaders": True,
                "isActive": True,
            })
            self.assertEqual(status, 201)
            interface_id = source_interface["interfaceId"]

            status, metadata = api_request(
                "POST", f"/api/loader/interfaces/{interface_id}/metadata", {},
            )
            self.assertEqual(status, 200, metadata)
            self.assertEqual(metadata["object"]["objectName"], "sample")
            object_id = metadata["object"]["objectId"]
            self.assertEqual(
                [(column["columnName"], column["columnOrdinalPosition"]) for column in metadata["columns"]],
                [("name", 1), ("amount", 2), ("active", 3)],
            )
            self.assertEqual(
                [column["datatype"] for column in metadata["columns"]],
                ["VARCHAR(10)", "DECIMAL(6,2)", "BOOLEAN"],
            )

            status, loaded = api_request("POST", f"/api/loader/interfaces/{interface_id}/load", {})
            self.assertEqual(status, 200, loaded)
            self.assertEqual(loaded["rowsLoaded"], 2)
            table = loaded["tableName"]
            self.assertEqual(loaded_data_scalar(f"SELECT COUNT(*) FROM `{table}`"), "2")
            self.assertEqual(
                loaded_data_scalar(f"SELECT amount FROM `{table}` WHERE name = 'Acme; Inc.'"),
                "1234.50",
            )
            self.assertEqual(
                database_scalar(
                    "SELECT COUNT(*) FROM information_schema.tables "
                    f"WHERE table_schema = DATABASE() AND table_name = '{table}'"
                ),
                0,
            )

            file.write_text("changed;header\nignored;row\n", encoding="utf-8")
            status, failure = api_request("POST", f"/api/loader/interfaces/{interface_id}/load", {})
            self.assertEqual(status, 409)
            self.assertIn("header", failure["detail"])
            self.assertEqual(loaded_data_scalar(f"SELECT COUNT(*) FROM `{table}`"), "2")
        finally:
            if object_id:
                _, columns = api_request("GET", "/api/metadata/columns")
                for column in columns:
                    if column["objectId"] == object_id:
                        api_request(
                            "DELETE",
                            f"/api/metadata/columns?{urlencode({'id': column['columnId']})}",
                        )
                api_request(
                    "DELETE",
                    f"/api/metadata/objects?{urlencode({'id': object_id})}",
                )
            if interface_id:
                api_request(
                    "DELETE",
                    f"/api/metadata/interfaces?{urlencode({'id': interface_id})}",
                )
            api_request(
                "DELETE",
                f"/api/metadata/source-systems?{urlencode({'id': source_name})}",
            )
            if table:
                database_admin_execute(f"DROP TABLE IF EXISTS loaded_data.`{table}`")
            file.unlink(missing_ok=True)

    def test_flatfile_loader_reports_missing_files_and_invalid_headers(self):
        suffix = uuid.uuid4().hex
        source_name = f"loader-failure-source-{suffix}"
        interface_id = None
        try:
            status, _ = api_request("POST", "/api/metadata/source-systems", {
                "sourceName": source_name,
                "sourceFunctionalName": "CSV source",
                "sourceType": "flatfile",
                "sourceOwner": "Integration",
                "sourceApplication": "Mimir test",
                "sourceLocation": "/data",
                "isActive": True,
            })
            self.assertEqual(status, 201)
            status, source_interface = api_request("POST", "/api/metadata/interfaces", {
                "interfaceName": "missing",
                "sourceName": source_name,
                "authenticationId": None,
                "interfaceFunctionalName": "Missing CSV",
                "interfaceType": "csv",
                "interfaceLocation": f"/data/missing-{suffix}.csv",
                "decimalSeparator": ".",
                "thousandsSeparator": ",",
                "delimiter": ";",
                "hasHeaders": True,
                "isActive": True,
            })
            self.assertEqual(status, 201)
            interface_id = source_interface["interfaceId"]
            status, failure = api_request(
                "POST", f"/api/loader/interfaces/{interface_id}/metadata", {},
            )
            self.assertEqual(status, 400)
            self.assertIn("cannot be read", failure["detail"])
        finally:
            if interface_id:
                api_request(
                    "DELETE",
                    f"/api/metadata/interfaces?{urlencode({'id': interface_id})}",
                )
            api_request(
                "DELETE",
                f"/api/metadata/source-systems?{urlencode({'id': source_name})}",
            )

    def test_loaded_data_database_is_separate_permissioned_and_persistent(self):
        suffix = uuid.uuid4().hex
        table_name = f"persistence_{suffix}"
        marker = f"loaded database persistence {suffix}"
        database_execute(
            "INSERT INTO application_log (occurred_at, log_level, source_name, message) "
            f"VALUES (CURRENT_TIMESTAMP, 'INFO', 'LoadedDataTest', '{marker}')"
        )
        self.addCleanup(
            database_execute,
            f"DELETE FROM application_log WHERE message = '{marker}'",
        )
        loaded_data_execute(
            f"CREATE TABLE `{table_name}` (value varchar(64) NOT NULL)"
        )
        self.addCleanup(
            database_admin_execute,
            f"DROP TABLE loaded_data.`{table_name}`",
        )
        loaded_data_execute(
            f"INSERT INTO `{table_name}` (value) VALUES ('{marker}')"
        )

        self.assertEqual(loaded_data_scalar(f"SELECT value FROM `{table_name}`"), marker)
        self.assertEqual(
            database_scalar(
                "SELECT COUNT(*) FROM information_schema.tables "
                f"WHERE table_schema = DATABASE() AND table_name = '{table_name}'"
            ),
            0,
        )
        self.assertEqual(
            database_scalar(
                "SELECT COUNT(*) FROM application_log "
                f"WHERE source_name = 'LoadedDataTest' AND message = '{marker}'"
            ),
            1,
        )
        self.assertNotEqual(
            loaded_data_scalar("SELECT DATABASE()"),
            database_query("SELECT DATABASE()"),
        )

        denied_metadata_access = subprocess.run(
            [
                "docker", "compose", "exec", "-T", "database", "sh", "-c",
                'MYSQL_PWD="$LOADED_DB_PASSWORD" mariadb --execute "$1" '
                '--database="$LOADED_DB_NAME" --user="$LOADED_DB_USER"',
                "mimir-loaded-data-query",
                "SELECT COUNT(*) FROM mimir.application_log",
            ],
            capture_output=True, text=True,
        )
        self.assertNotEqual(denied_metadata_access.returncode, 0)
        denied_loaded_data_access = subprocess.run(
            [
                "docker", "compose", "exec", "-T", "database", "sh", "-c",
                'mariadb --execute "$1" --database="$MARIADB_DATABASE" '
                '--user="$MARIADB_USER" --password="$MARIADB_PASSWORD"',
                "mimir-metadata-query",
                f"SELECT COUNT(*) FROM loaded_data.`{table_name}`",
            ],
            capture_output=True, text=True,
        )
        self.assertNotEqual(denied_loaded_data_access.returncode, 0)

        for _ in range(2):
            subprocess.run(["docker", "compose", "restart", "database"], check=True)
            subprocess.run(
                ["docker", "compose", "up", "--no-build", "--detach", "--wait",
                 "--wait-timeout", "180", "database"],
                check=True,
            )
            self.assertEqual(loaded_data_scalar(f"SELECT value FROM `{table_name}`"), marker)
            self.assertEqual(
                database_scalar(
                    "SELECT COUNT(*) FROM application_log "
                    f"WHERE source_name = 'LoadedDataTest' AND message = '{marker}'"
                ),
                1,
            )

        subprocess.run(["docker", "compose", "restart", "backend"], check=True)
        subprocess.run(
            ["docker", "compose", "up", "--no-build", "--detach", "--wait",
             "--wait-timeout", "180", "backend"],
            check=True,
        )
        self.assertEqual(
            json.loads(fetch("/api/status")[1]),
            {"application": "mimir", "database": "connected"},
        )
        self.assertEqual(loaded_data_scalar(f"SELECT value FROM `{table_name}`"), marker)

    def test_metadata_crud_relationships_and_persistence_after_restart(self):
        application_logs_before = database_scalar("SELECT COUNT(*) FROM application_log")
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
            "interfaceId": None,
            "objectFunctionalName": "Integration object",
            "objectDescription": "Integration description",
            "isConfidential": False,
            "isBusinessCritical": True,
            "isActive": True,
        }
        api_parameter = {
            "parameterId": f"parameter-{suffix}",
            "interfaceId": None,
            "parameterName": "api-version",
            "parameterValue": "v1",
        }
        column = {
            "columnName": f"column-{suffix}",
            "objectId": None,
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
            ("interfaces", "interfaceId", interface),
            ("api-parameters", "parameterId", api_parameter),
            ("objects", "objectId", source_object),
            ("columns", "columnId", column),
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
                self.assertEqual(status, 201)
                saved = dict(value)
                if resource in ("interfaces", "objects", "columns"):
                    saved[key] = body[key]
                created.append((resource, key, saved))
                self.assertEqual(body, saved)
                self.assertEqual(request("GET", resource, identifier=saved[key])[1], saved)
                self.assertIn(saved, request("GET", resource)[1])
                with self.assertRaises(HTTPError) as duplicate:
                    request("POST", resource, value)
                self.assertEqual(duplicate.exception.code, 409)
                if resource == "interfaces":
                    interface["interfaceId"] = saved["interfaceId"]
                    api_parameter["interfaceId"] = saved["interfaceId"]
                    source_object["interfaceId"] = saved["interfaceId"]
                if resource == "objects":
                    source_object["objectId"] = saved["objectId"]
                    column["objectId"] = saved["objectId"]

            other_source = dict(source)
            other_source["sourceName"] += "-other"
            status, other_source = request("POST", "source-systems", other_source)
            self.assertEqual(status, 201)
            created.append(("source-systems", "sourceName", other_source))

            other_interface = dict(interface)
            other_interface["sourceName"] = other_source["sourceName"]
            status, other_interface = request("POST", "interfaces", other_interface)
            self.assertEqual(status, 201)
            created.append(("interfaces", "interfaceId", other_interface))
            self.assertEqual(other_interface["interfaceName"], interface["interfaceName"])
            with self.assertRaises(HTTPError) as duplicate_interface:
                request("POST", "interfaces", interface)
            self.assertEqual(duplicate_interface.exception.code, 409)

            other_object = dict(source_object)
            other_object["interfaceId"] = other_interface["interfaceId"]
            status, other_object = request("POST", "objects", other_object)
            self.assertEqual(status, 201)
            created.append(("objects", "objectId", other_object))
            self.assertEqual(other_object["objectName"], source_object["objectName"])
            with self.assertRaises(HTTPError) as duplicate_object:
                request("POST", "objects", other_object)
            self.assertEqual(duplicate_object.exception.code, 409)

            other_column = dict(column)
            other_column["objectId"] = other_object["objectId"]
            status, other_column = request("POST", "columns", other_column)
            self.assertEqual(status, 201)
            created.append(("columns", "columnId", other_column))
            self.assertEqual(other_column["columnName"], column["columnName"])
            with self.assertRaises(HTTPError) as duplicate_column:
                request("POST", "columns", other_column)
            self.assertEqual(duplicate_column.exception.code, 409)

            other_parameter = dict(api_parameter)
            other_parameter["parameterId"] += "-other"
            other_parameter["interfaceId"] = other_interface["interfaceId"]
            status, other_parameter = request("POST", "api-parameters", other_parameter)
            self.assertEqual(status, 201)
            created.append(("api-parameters", "parameterId", other_parameter))
            self.assertEqual(
                request("GET", "api-parameters", identifier=other_parameter["parameterId"])[1]["interfaceId"],
                other_interface["interfaceId"],
            )

            for resource, key, value in created:
                if resource in ("api-parameters", "columns"):
                    continue
                with self.assertRaises(HTTPError) as referenced:
                    request("DELETE", resource, identifier=value[key])
                self.assertEqual(referenced.exception.code, 409)

            _, audit_json = fetch("/api/audit")
            audit_entries = json.loads(audit_json)
            self.assertTrue(any(entry["operation"] == "CREATE" for entry in audit_entries))
            self.assertTrue(any("[REDACTED]" in entry["details"] for entry in audit_entries))
            self.assertNotIn("local-test-credential", audit_json)
            self.assertNotIn('"parameterValue":"v1"', audit_json)
            persisted_log_count = database_scalar("SELECT COUNT(*) FROM application_log")
            metadata_log_count = database_scalar(
                "SELECT COUNT(*) FROM application_log WHERE source_name = 'MetadataStore'"
            )
            self.assertGreater(persisted_log_count, application_logs_before)

            subprocess.run(
                ["docker", "compose", "restart", "backend"], check=True,
            )
            subprocess.run(
                ["docker", "compose", "up", "--no-build", "--no-recreate", "--detach", "--wait",
                 "--wait-timeout", "180", "backend"],
                check=True,
            )
            self.assertGreaterEqual(
                database_scalar("SELECT COUNT(*) FROM application_log"), persisted_log_count,
            )
            self.assertEqual(
                database_scalar(
                    "SELECT COUNT(*) FROM application_log WHERE source_name = 'MetadataStore'"
                ),
                metadata_log_count,
            )
            _, audit_after_restart = fetch("/api/audit")
            self.assertEqual(json.loads(audit_after_restart), audit_entries)
            for resource, key, value in created:
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

    def test_upgrade_migrates_existing_mariadb_hierarchy_without_losing_audit(self):
        subprocess.run(["docker", "compose", "stop", "backend"], check=True)
        database_execute(
            """
            ALTER TABLE api_parameter DROP FOREIGN KEY fk_api_parameter_interface_id;
            ALTER TABLE source_column DROP FOREIGN KEY fk_column_object_id;
            ALTER TABLE source_object DROP FOREIGN KEY fk_object_interface_id;
            ALTER TABLE api_parameter DROP INDEX idx_api_parameter_interface_id;
            ALTER TABLE source_column DROP INDEX idx_column_object_id;
            ALTER TABLE source_object DROP INDEX idx_object_interface_id;
            ALTER TABLE source_system_interface DROP INDEX uq_interface_source_name;
            ALTER TABLE source_object DROP INDEX uq_object_interface_name;
            ALTER TABLE source_column DROP INDEX uq_column_object_name;
            ALTER TABLE source_system_interface DROP PRIMARY KEY;
            ALTER TABLE source_object DROP PRIMARY KEY;
            ALTER TABLE source_column DROP PRIMARY KEY;
            ALTER TABLE source_object ADD COLUMN interface_name varchar(255);
            ALTER TABLE source_column ADD COLUMN object_name varchar(255);
            ALTER TABLE source_system_interface DROP COLUMN interface_id;
            ALTER TABLE source_object DROP COLUMN interface_id, DROP COLUMN object_id;
            ALTER TABLE source_column DROP COLUMN object_id, DROP COLUMN column_id;
            ALTER TABLE source_system_interface ADD PRIMARY KEY (interface_name);
            ALTER TABLE source_object ADD PRIMARY KEY (object_name);
            ALTER TABLE source_column ADD PRIMARY KEY (column_name);
            ALTER TABLE source_object ADD CONSTRAINT fk_object_interface
              FOREIGN KEY (interface_name) REFERENCES source_system_interface(interface_name);
            ALTER TABLE source_column ADD CONSTRAINT fk_column_object
              FOREIGN KEY (object_name) REFERENCES source_object(object_name);
            ALTER TABLE api_parameter ADD CONSTRAINT fk_api_parameter_interface
              FOREIGN KEY (interface_id) REFERENCES source_system_interface(interface_name);
            CREATE INDEX idx_object_interface ON source_object(interface_name);
            CREATE INDEX idx_column_object ON source_column(object_name);
            CREATE INDEX idx_api_parameter_interface ON api_parameter(interface_id);
            DELETE FROM DATABASECHANGELOG WHERE ID = '005-scoped-metadata-names';
            INSERT INTO source_system (source_name, source_functional_name, source_type,
              source_owner, source_application, source_location, is_active)
              VALUES ('migration-source', 'Migration source', 'database', NULL, 'test', NULL, TRUE);
            INSERT INTO source_system_interface (interface_name, source_name, authentication_id,
              interface_functional_name, interface_type, interface_location, decimal_separator,
              thousands_separator, delimiter, has_headers, is_active)
              VALUES ('migration-interface', 'migration-source', NULL, 'Migration interface',
              'jdbc', 'jdbc:test', '.', ',', NULL, TRUE, TRUE);
            INSERT INTO source_object (object_name, interface_name, object_functional_name,
              object_description, is_confidential, is_business_critical, is_active)
              VALUES ('migration-object', 'migration-interface', 'Migration object', NULL,
              FALSE, FALSE, TRUE);
            INSERT INTO source_column (column_name, object_name, column_functional_name, datatype,
              column_alternate_name, column_description, column_ordinal_position, needs_encryption,
              is_partition_field, partition_ordinal_position, is_key, key_ordinal_position,
              is_nullable, test_datatype, character_maximum_length, numeric_scale,
              numeric_precision, null_value_regex_pattern, is_active)
              VALUES ('migration-column', 'migration-object', 'Migration column', 'varchar',
              NULL, NULL, 1, FALSE, FALSE, 0, FALSE, 0, TRUE, FALSE, 255, NULL, NULL, NULL, TRUE);
            INSERT INTO api_parameter (parameter_id, interface_id, parameter_name, parameter_value)
              VALUES ('migration-parameter', 'migration-interface', 'version', 'v1');
            INSERT INTO metadata_audit (occurred_at, operation, entity_type, record_id, details)
              VALUES (CURRENT_TIMESTAMP, 'CREATE', 'SourceObject', 'migration-object',
              '{"objectName":"migration-object"}');
            """
        )
        subprocess.run(
            ["docker", "compose", "up", "--no-build", "--detach", "--wait",
             "--wait-timeout", "180", "backend"],
            check=True,
        )
        self.assertEqual(
            database_scalar(
                "SELECT COUNT(*) FROM source_column WHERE column_id = 'migration-column' "
                "AND object_id = 'migration-object'"
            ),
            1,
        )
        self.assertEqual(
            database_scalar(
                "SELECT COUNT(*) FROM api_parameter WHERE parameter_id = 'migration-parameter' "
                "AND interface_id = 'migration-interface'"
            ),
            1,
        )
        _, response = fetch("/api/metadata/columns?id=migration-column")
        self.assertEqual(json.loads(response)["objectId"], "migration-object")
        _, audit = fetch("/api/audit")
        self.assertIn('"recordId":"migration-object"', audit)
        subprocess.run(["docker", "compose", "restart", "backend"], check=True)
        subprocess.run(
            ["docker", "compose", "up", "--no-build", "--detach", "--wait",
             "--wait-timeout", "180", "backend"],
            check=True,
        )
        self.assertEqual(
            database_scalar(
                "SELECT COUNT(*) FROM source_column WHERE column_id = 'migration-column' "
                "AND object_id = 'migration-object'"
            ),
            1,
        )


if __name__ == "__main__":
    unittest.main()
