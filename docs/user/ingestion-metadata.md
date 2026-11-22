# Using the ingestion metadata editor

## Start and navigate

Complete the [local setup](../technical/setup.md), then run `make run`.
Open `http://localhost:8080` (or the configured frontend port). The page shows
database connectivity and an **Ingestion metadata** editor. Choose **Source
systems**, **Authentications**, **Interfaces**, **Objects**, or **Columns** to
view all records of that kind.

An empty view displays **No records yet**. **Refresh** reloads all five lists
and relationship choices. Loading or writes temporarily disable actions.
If loading fails, the editor displays an error and disables creation until
Refresh succeeds.

## Create a metadata hierarchy

1. Choose **Source systems**, then **New record**. Enter a globally unique source
   name, its business-facing functional name, and a source type; select **Save**.
2. If an interface needs authentication, create an **Authentication** first.
   Enter an id, type, and the location of the authentication material, such as
   `vault://team/crm`. **Never enter passwords, tokens, or credential contents**.
3. Under **Interfaces**, create a record with a unique interface name. Select
   its source system and optional authentication from the lists. **None**
   means no authentication reference. Enter its functional name, type, and
   access location.
4. Under **Objects**, create an object with a unique object name and functional
   name, selecting its interface.
5. Under **Columns**, create a column with a unique column name, functional
   name, and datatype, selecting its object.

Fields marked `*` are required and cannot be whitespace-only. Names, types,
functional names, datatypes, and references allow up to 255 characters.
Interface and authentication locations allow up to 2048 characters. Types are
free text. Interface, object, and column names are globally unique within their
entity, **not** just within their parent.

Save remains disabled until the fields are valid. A successful save displays
**Record saved** and reloads the lists. **Cancel** discards an unsaved form.
There is no automatic save.

## Edit and delete

Select a row's **Edit** button to change its fields and relationships. The
primary key is read-only. To use another key, create a new record, move its
dependent records, and delete the old record. Select **Save** to persist edits.
Concurrent edits are last-write-wins; refresh before editing when others may
be working on the same metadata.

Select **Delete**, review the named record, and select **Confirm delete**.
**Cancel** leaves it unchanged. Referenced records cannot be deleted. Remove
or reassign dependent columns first, then objects, then interfaces, before
deleting a source system or authentication.

## Recover from errors

- **Conflict**: check for a duplicate key or a missing parent. For deletion,
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
