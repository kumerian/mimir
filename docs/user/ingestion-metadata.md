# Using the ingestion metadata editor

## Start and navigate

Complete the [local setup](../technical/setup.md), then run `make run`.
Open `http://localhost:8080` (or the configured frontend port). The page shows
the Mimir banner with its 32x32 logo, and the browser tab uses the Mimir favicon.
Below the banner are
database connectivity and an **Ingestion metadata** editor. Choose **Source
systems**, **Authentications**, **API parameters**, **Interfaces**, **Objects**, or **Columns** to
view all records of that kind.

An empty view displays **No records yet**. **Refresh** reloads all six lists
and relationship choices. Loading or writes temporarily disable actions.
If loading fails, the editor displays an error and disables creation until
Refresh succeeds.

## Create a metadata hierarchy

1. Choose **Source systems**, then **New record**. Enter a unique source
   name, source type, application, and active state. Optionally enter the functional
   name, owner, and location; select **Save**.
2. If an interface needs authentication, create an **Authentication** first.
   Enter an id, type, location of the authentication material, such as
   `vault://team/crm`, and active state. The optional user and credential fields
   are stored unencrypted and returned by the unauthenticated API; use only trusted
   local-development values, never production secrets.
3. Under **Interfaces**, create a record with a name unique within its source
   system. Select its source system and optional authentication from the lists. **None**
   means no authentication reference. Enter its functional name, type, access location, decimal and thousands
   separators, optional delimiter/header setting, and active state.
   For a CSV file, use interface type `CSV` or `Flatfile`, set **Has headers** to
   true, and enter its backend path such as `/data/customers.csv`. By default,
   Compose mounts the host's `./flatfiles` directory read-only at `/data`; change
   `FLATFILE_DIRECTORY` in `.env` to use another host folder.
4. For an API interface, create **API parameters** by entering a unique parameter
   id and name, selecting its interface by its source/interface context, and optionally entering a value. Values
   are stored unencrypted and returned by the unauthenticated API; never use
   production secrets.
5. Under **Objects**, create an object with a name unique within its interface,
selecting the interface by its source/interface context. Set confidentiality, business-critical, and active states; optionally
enter a functional name and description.
6. Under **Columns**, create a column with a name unique within its object,
datatype, ordinal position, encryption, partition, key, nullability, datatype-testing,
and active settings, selecting the object by its full source/interface/object context. Optional fields include alternate
name, functional name, description, ordinals, length, numeric scale and precision,
and a null-value regex. Names may be reused in other parent hierarchies, and the
editor shows the full hierarchy in lists and parent selectors.

## Derive metadata and load a CSV

After saving a CSV/Flatfile Interface, choose **Details** in its Interfaces row.
Select **Create/update Object and Columns** to read the header and data rows.
The Object name is set to the Interface name. Columns follow header order; the
loader trims header whitespace, removes a UTF-8 BOM from the first header, and
rejects blank, duplicate, or overlength names and inconsistent row widths.
Datatype inference recognizes booleans, integers, decimals using the Interface's
decimal/thousands separators, ISO dates, and ISO local date-times; uncertain
values use `VARCHAR`, while text too long for `VARCHAR` uses `MEDIUMTEXT`.
Repeating metadata derivation updates matching Columns, adds new ones, and
removes Columns no longer present while preserving existing Column settings
that are not inferred.

Select **Load CSV data** to append data rows to the loaded-data database. The
header is not loaded. Empty cells become SQL `NULL`. The Object's stable ID
determines its physical destination table name, avoiding collisions between
same-named Objects under different Sources. Tables are created from saved Column
metadata and rows are written transactionally. If a table already exists, its
Column order, names, datatypes, and nullability must still match; schema drift
is rejected. Repeated completed loads append another copy of the file's rows.
The buttons are disabled while a request is running, and the backend rejects a
second concurrent load for the same Interface. Failed loads report that a table
may already have been created; data rows from the failed load are rolled back.

Only CSV/Flatfile interfaces with a header are supported. CSV files are read as
UTF-8 using the configured delimiter (comma when unset) and standard double-quote
escaping. Locations must be file paths accessible to the backend container;
host files outside `FLATFILE_DIRECTORY` are not mounted by default. Remote URLs,
headerless files, schema drift, and safe-access controls are not supported yet.

Fields marked `*` are required and cannot be whitespace-only. Names, types,
functional names, datatypes, and references allow up to 255 characters.
Interface and authentication locations allow up to 2048 characters. Types are
free text. Interface names are unique within a source system, object names within
an interface, and column names within an object. Stable IDs are generated for
interfaces, objects, and columns; changing a name or parent does not change
these IDs or break dependent relationships. Source names remain globally unique.

Save remains disabled until the fields are valid. A successful save displays
**Record saved** and reloads the lists. **Cancel** discards an unsaved form.
There is no automatic save.

## Edit and delete

Select a row's **Edit** button to change its fields and relationships. Source
names cannot be changed; IDs for interfaces, objects, and columns are stable
and not editable, while their names can be changed when the destination name is
available. Reparenting preserves dependent records, and conflicts leave the
record in its original hierarchy. Select **Save** to persist edits.
Concurrent edits are last-write-wins; refresh before editing when others may
be working on the same metadata.

Select **Delete**, review the named record, and select **Confirm delete**.
**Cancel** leaves it unchanged. Referenced records cannot be deleted. Remove
or reassign dependent columns, API parameters, and objects before deleting
their parent records. Remove or reassign interfaces before deleting a source
system or authentication.

## Recover from errors

- **Conflict**: check for a duplicate name under the selected parent or a missing parent. For deletion,
  remove or reassign dependent records. Cancel the form/confirmation and
  Refresh to retrieve changes made by another user.
- **Invalid data**: check required fields, lengths, and the unchanged key.
- **Record no longer exists**: cancel and Refresh before creating or editing.
- **Cannot reach the backend**: check the stack, then retry or Refresh.
  A failed response does not confirm whether a write reached the server;
  check the refreshed list before creating the same key again.

Failed writes keep the entered values available for correction and retry.
If saving succeeds but the subsequent reload fails, **Record saved** remains
visible alongside the loading error; use Refresh rather than saving again.

Data persists in the MariaDB volume across application restarts and `make down`.
`make run` starts it again without deleting metadata. This editor manages
ingestion metadata; it does not fetch source data or execute ingestion jobs.
