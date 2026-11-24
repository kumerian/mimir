# Create a flatfile loader

## Desired outcome

A user can derive Object and Column metadata from a flatfile while entering an Interface and trigger the loading of that file into the separate loaded-data database from the Interface details page.

## Scope

Add a web-interface function during Interface entry that reads the flatfile at the Interface location and creates the corresponding Object and Column records. Add a function on the Interface details page that loads data from that location into the loaded-data database using the Interface and its metadata. Include the backend and file access needed for both functions.

## Acceptance criteria

- [ ] During entry of a flatfile Interface, the user can invoke a function that reads the header at the entered Interface location and creates an Object linked to the Interface, with an Object Name equal to the Interface name.
- [ ] The function creates a Column linked to that Object for each header field, with its name and ordinal position reflecting the header and its order. The resulting records satisfy the existing metadata model and are visible in the editor.
- [ ] The Interface details page offers a function to load the flatfile at that Interface's location. Triggering it writes the file's data rows, associated with the corresponding Object and Columns, to the separate loaded-data database; the header is not loaded as a data row.
- [ ] The user can distinguish successful metadata creation and loading from failures. Missing or unreadable files, invalid headers, metadata name conflicts, and load or database errors are surfaced without claiming success; any partial effects are made clear.
- [ ] Existing metadata remains intact when creation encounters a name conflict, and existing metadata CRUD continues to work.
- [ ] Focused tests cover header-derived metadata and ordering, a successful load with its resulting data, and relevant failure cases. Tests exercise the web interface and backend, including the loaded-data database; directly related user and setup documentation is updated.

## Dependencies and context

- Depends on [Set up a separate database for loaded data](loaded-data-database.md).
- The current editor manages metadata only; locations are not yet accessed. Interface names are unique within a Source, Object names within an Interface, and Column names within an Object. A Column has required fields beyond its header name and position. An Object references an Interface ID, and a Column references an Object ID.
- [Scope metadata names to their parent hierarchy](../metadata/scoped-metadata-names.md) is complete; generated records must follow the scoped uniqueness rules.
- The Interface includes delimiter, header, decimal separator, and thousands separator settings. The current containerized backend must be able to access the specified file location; the browser's local filesystem is not automatically available to it.

## Open questions

- Which flatfile formats, encodings, quoting rules, and location types are supported, and how are the Interface settings applied?
- What should happen for Interfaces without headers, empty or duplicate header names, or names that conflict with the applicable metadata uniqueness rules?
- How should required Object and Column fields not supplied by the header, including datatype, be determined or entered?
- How should the creation function handle an unsaved Interface and repeated invocation, and when should generated records be persisted?
- How are destination tables and column types selected or created, and what are the policies for repeated loads, existing data, invalid rows, and partial failures?
- What file-access restrictions and deployment configuration are needed for safe access, given that the current application has no login or authorization layer?
