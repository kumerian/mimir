# mimir
Metadata-based data ingestion toolset

Java/Spring Boot backend, Angular frontend, and MariaDB with Liquibase-managed
ingestion metadata and a CRUD editor.

All setup, development, testing, and CI documentation is in
[docs/technical/setup.md](docs/technical/setup.md).

Start all services with `make run`; run all test suites with `make test`.
See the [metadata model and API](docs/technical/ingestion-metadata.md) and
[frontend user guide](docs/user/ingestion-metadata.md).
