# Create a flatfile loader

## Desired outcome

A user can derive Object and Column metadata from a flatfile while entering an Interface and trigger the loading of that file into the separate loaded-data database from the Interface details page.

## Scope

Add a web-interface function during Interface entry that reads the flatfile at the Interface location and creates the corresponding Object and Column records. Add a function on the Interface details page that loads data from that location into the loaded-data database using the Interface and its metadata. Include the backend and file access needed for both functions.

## Acceptance criteria

- [x] During entry of a flatfile Interface, the user can invoke a function that reads the header at the saved Interface location and creates an Object linked to the Interface, with an Object Name equal to the Interface name.
- [x] The function creates a Column linked to that Object for each header field, with its name and ordinal position reflecting the header and its order. The resulting records satisfy the existing metadata model and are visible in the editor.
- [x] The Interface details page offers a function to load the flatfile at that Interface's location. Triggering it writes the file's data rows, associated with the corresponding Object and Columns, to the separate loaded-data database; the header is not loaded as a data row.
- [x] The user can distinguish successful metadata creation and loading from failures. Missing or unreadable files, invalid headers, metadata conflicts, and load or database errors are surfaced without claiming success; partial load effects are explained.
- [x] Existing metadata remains intact when metadata derivation encounters a conflict, and existing metadata CRUD continues to work.
- [x] Focused tests cover header-derived metadata and ordering, a successful load with its resulting data, and relevant failure cases. Tests exercise the web interface and backend, including the loaded-data database; directly related user and setup documentation is updated.

## Dependencies and context

- Depends on [Set up a separate database for loaded data](loaded-data-database.md).
- The current editor manages metadata only; locations are not yet accessed. Interface names are unique within a Source, Object names within an Interface, and Column names within an Object. A Column has required fields beyond its header name and position. An Object references an Interface ID, and a Column references an Object ID.
- [Scope metadata names to their parent hierarchy](../metadata/scoped-metadata-names.md) is complete; generated records must follow the scoped uniqueness rules.
- The Interface includes delimiter, header, decimal separator, and thousands separator settings. The current containerized backend must be able to access the specified file location; the browser's local filesystem is not automatically available to it.

## Decisions and limitations

- Only saved CSV/Flatfile Interfaces with headers are supported. Files are UTF-8, use the configured delimiter (comma by default) and standard double-quote escaping, and must be accessible to the backend as a file path. Headerless files and remote locations are out of scope.
- Header whitespace is trimmed and a UTF-8 BOM is removed from the first field. Blank, duplicate, overlength headers and rows with inconsistent field counts are rejected.
- Datatypes are inferred from all non-empty values: booleans, integers, decimals using the Interface's separators, ISO dates, and ISO local date-times are recognized. Uncertain text uses `VARCHAR`; text exceeding its supported length uses `MEDIUMTEXT`. Empty cells load as SQL `NULL`.
- Metadata derivation is available from the saved Interface details page, and its action is disabled while a request is running. Repeated derivation reconciles the Interface-named Object and its Columns, preserving matching IDs and non-inferred settings, adding new Columns, and removing obsolete Columns. Ambiguous existing Objects are rejected without changing metadata.
- Loading creates the destination table from saved metadata when needed and appends rows. A second concurrent load for the same Interface is rejected; schema drift is not supported. Data rows are transactional and roll back on failure, but a newly created destination table may remain.
- Compose mounts `FLATFILE_DIRECTORY` read-only at `/data`. Safe-access controls and authentication are not implemented; this is not suitable for an untrusted or public deployment.

## Verification

- Focused backend tests: `.\mvnw.cmd '-Dtest=FlatfileLoaderIntegrationTest,FlatfileLoaderServiceTest' test`.
- Frontend production build: `npm --prefix frontend run build`.
- Python test-runner tests: `python -m unittest discover -s tests -p test_runner.py -v`.
- Full suite: `make test` passed (40 backend tests, 20 Angular tests, and 10 Docker integration tests). The frontend production build passed, as did all 7 Python test-runner tests.
