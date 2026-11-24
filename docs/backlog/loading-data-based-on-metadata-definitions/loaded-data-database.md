# Set up a separate database for loaded data

## Desired outcome

Data ingested by load functionality has durable storage in a database separate from the metadata database, on the same database server.

## Scope

Set up the loaded-data database and the application connectivity and permissions needed by loaders. Support both new installations and existing installations without resetting their metadata. The current stack uses MariaDB, a persistent volume, and Liquibase-managed metadata; this item does not implement a loader.

## Acceptance criteria

- [ ] A new installation provides a database for loaded data distinct from the metadata database on the same MariaDB server.
- [ ] An existing installation can obtain the loaded-data database without losing or recreating its metadata or existing database volume; repeating setup preserves both databases' existing contents.
- [ ] The application can connect to and use the loaded-data database for loader reads and writes with the necessary permissions; metadata operations continue to use the metadata database.
- [ ] Data written to the loaded-data database remains available after an application and database restart, and is not stored in the metadata database.
- [ ] Setup or connection failures are surfaced rather than reported as success.
- [ ] Focused tests cover fresh and existing installations, database separation, connectivity, persistence, and a failure case; setup and configuration documentation is updated.

## Dependencies and open questions

- This database is a prerequisite for the [flatfile loader](flatfile-loader.md).
- The database name, configuration, provisioning mechanism, and account/permission model need to be decided without assuming production credentials or privileges.
