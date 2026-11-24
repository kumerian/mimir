# Set up a separate database for loaded data

## Desired outcome

Data ingested by load functionality has durable storage in a database separate from the metadata database, on the same database server.

## Scope

Set up the loaded-data database and the application connectivity and permissions needed by loaders. Support both new installations and existing installations without resetting their metadata. The current stack uses MariaDB, a persistent volume, and Liquibase-managed metadata; this item does not implement a loader.

## Acceptance criteria

- [x] A new installation provides a database for loaded data distinct from the metadata database on the same MariaDB server.
- [x] An existing installation can obtain the loaded-data database without losing or recreating its metadata or existing database volume; repeating setup preserves both databases' existing contents.
- [x] The application can connect to and use the loaded-data database for loader reads and writes with the necessary permissions; metadata operations continue to use the metadata database.
- [x] Data written to the loaded-data database remains available after an application and database restart, and is not stored in the metadata database.
- [x] Setup or connection failures are surfaced rather than reported as success.
- [x] Focused tests cover fresh and existing installations, database separation, connectivity, persistence, and a failure case; setup and configuration documentation is updated.

## Decisions and follow-up

- This database is a prerequisite for the [flatfile loader](flatfile-loader.md).
- The database defaults to `loaded_data`, configured by `LOADED_DB_NAME`; the distinct application account defaults to `mimir_loader`, configured by `LOADED_DB_USER` and `LOADED_DB_PASSWORD`.
- A MariaDB entrypoint provisions the schema and grants idempotently at every start, supporting fresh volumes and existing volumes without resetting data. The metadata and loaded-data datasource URLs, accounts, and passwords are configured independently.
- The loader account is limited to the loaded-data schema with read/write, create/alter, index, and temporary-table privileges; it receives no metadata access or `DROP` privilege.
- The [flatfile loader](flatfile-loader.md) remains the follow-up that will use this database.
- Verification: `make test` passed, including Java connection-failure and datasource-separation tests and Docker checks for fresh provisioning, denied cross-database access, data persistence through repeated database and backend restarts, and unchanged metadata.
