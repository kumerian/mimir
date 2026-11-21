# Local reference ingestion

## Status and scope

The local reference profile is an implemented development/test capability.
It is not a production deployment, a database adapter, an identity provider, or
a claim of cross-system exactly-once delivery. It supports one active metadata
revision, one bounded JSON Lines source object, the local append-only
destination described in the [operations guide](../operations/local-reference.md),
finite transformation and quality-rule catalogs, persisted quarantine, and
explicit separately invoked cleaning. Schedule evaluation creates durable run
requests but does not start ingestion.

## Prerequisites and permissions

- Go 1.27.4 to build from source; `make` and Python 3.12.3 for repository
  tasks (see `.python-version`).
- A local directory containing synthetic or otherwise locally authorized
  records. The reference profile permits only `public` and `internal`
  classifications.
- A caller who explicitly selects the local development profile with
  `--allow-local`. This is a local-use guard, not production authorization.

No production credentials or network services are used. Draft metadata may
contain opaque `env://NAME` references, but local file adapters do not consume
them: activation, discovery, and execution reject such references with
`unsupported_capability` rather than silently ignoring them. A local-only resolver port
implementation accepts that reference form for a bounded synchronous callback
with a five-second operation context; it is not wired to user metadata or
enabled as a production secret provider. The callback must honor its deadline
and must not retain or copy the value. Credential resolution in an authorized
adapter operation and secret expiry are not implemented. Do not put secret
values in metadata or input records.
Previously activated local revisions with secret references can no longer run
or be discovered; author and activate a new credential-free revision. Retain
the old revision and state for provenance; no automatic migration occurs.

Activation checks the effective classification of every object and column;
only public and internal are accepted. Discovery and execution recheck this
policy against already-active revisions, so a previously activated revision
outside the local ceiling cannot be read or published. Missing or unknown
classification is invalid. The local grant is not a hosted identity policy.

## Metadata v1

Metadata is strict JSON. Duplicate keys, unknown fields, unsupported versions,
invalid identifiers, duplicate object/column IDs, path traversal, and invalid
classifications fail closed. Revision/source/object/column/rule and secret
reference IDs must be 1–128 ASCII letters, digits, dot, underscore, or hyphen,
starting with a letter or digit. Unknown field diagnostics name a safe recognized
field; field names outside the diagnostic allowlist are redacted. YAML is not accepted. The JSON Schema is
[`contracts/metadata/v1.schema.json`](../../contracts/metadata/v1.schema.json);
the pinned Draft 2020-12 validator and executable parser also perform semantic
validation. The schema is embedded in the binary and freshness-checked against
the contract source. `draft` metadata has no execution authority. Only the
`activate` use case creates an immutable, audited active revision.

`validate --metadata <file>` prints a deterministic validation report. Add
`--state-dir <private-dir>` to persist the immutable report; retrieve it later
with `validation-report --state-dir <private-dir> --input-digest <sha256>`.
Reports identify exact input bytes by digest and contain safe codes, paths,
messages, security impact, and compatibility decisions only; raw authoring
input and secret values are not retained. Every report has
`runEligible: false`. `author` stores the report before storing a draft, and
invalid metadata creates no revision. The local report store is append-only,
private to the local user, and supports lookup by exact input digest.

Synthetic candidate:

```json
{
  "schemaVersion": "mimir.metadata/v1",
  "revision": "orders-v1",
  "status": "draft",
  "sources": [{
    "id": "orders",
    "classification": "internal",
    "objects": [{"id": "orders", "path": "orders.jsonl"}]
  }]
}
```

The schema and parser reject unknown fields rather than silently permitting
forward compatibility. There is no v0 migration. Revisions are immutable;
rollback to prior metadata is an explicit activation of a retained revision.

## First local run

From the repository root on macOS or Linux:

```sh
make install-smoke
make test
make lint
make build
./bin/mimir validate --metadata contracts/examples/metadata-v1-draft.json
./bin/mimir validate --metadata contracts/examples/metadata-v1-draft.json \
  --state-dir /tmp/mimir-state
mkdir -p /tmp/mimir-source
printf '%s\n' '{"order_id":1,"customer_name":"Example"}' > /tmp/mimir-source/orders.jsonl
./bin/mimir activate \
  --metadata contracts/examples/metadata-v1-draft.json \
  --state-dir /tmp/mimir-state \
  --allow-local
./bin/mimir run \
  --source-root /tmp/mimir-source \
  --state-dir /tmp/mimir-state \
  --allow-local
./bin/mimir health
```

From PowerShell on Windows:

```powershell
make install-smoke
make test
make lint
make build
& .\bin\mimir.exe validate --metadata contracts\examples\metadata-v1-draft.json
$stateDir = Join-Path $env:TEMP 'mimir-state'
$sourceDir = Join-Path $env:TEMP 'mimir-source'
New-Item -ItemType Directory -Force -Path $sourceDir | Out-Null
& .\bin\mimir.exe validate `
  --metadata contracts\examples\metadata-v1-draft.json `
  --state-dir $stateDir
'{"order_id":1,"customer_name":"Example"}' |
  Set-Content -Encoding utf8 (Join-Path $sourceDir 'orders.jsonl')
& .\bin\mimir.exe activate `
  --metadata contracts\examples\metadata-v1-draft.json `
  --state-dir $stateDir `
  --allow-local
& .\bin\mimir.exe run `
  --source-root $sourceDir `
  --state-dir $stateDir `
  --allow-local
& .\bin\mimir.exe health
```

`activate` validates the candidate and records a new active revision; `run`
pins that active revision and a content-addressed source snapshot. Run output
contains only run/revision identity and record count. Raw record values are
written only to the local destination or quarantine journal, not to audit
output. Drafts can also be added and listed with `author` and `list-metadata`;
`discover` stores a schema proposal without changing active metadata.

The JSON Schema validator is a pinned module dependency. Its schema is embedded,
so installed binaries do not require a separate schema file or network access.
`make docs` checks schema syntax, contract version, example version, and
byte-for-byte freshness of the embedded schema. After changing the canonical
contract, run `make generate-metadata-schema` and then `make docs`. The CLI
report command exits nonzero for invalid input after writing its report to
standard output; when
`--state-dir` is supplied, the report is persisted before returning.

## Inputs, outputs, and guarantees

- Input is one source object represented as JSON Lines, up to 16 MiB and 1,000
  records; each record is a JSON object no larger than 256 KiB. Duplicate JSON
  keys are rejected.
- Metadata object paths must resolve beneath the configured source root,
  including through symlinks.
- The destination journal commits a stable batch identity. Repeating the same
  active metadata and source snapshot reconciles to the existing commit rather
  than appending duplicate data.
- The local source has no acknowledgement capability. The order is durable
  destination commit, durable checkpoint, then required commit audit. A run is
  not reported as successful before these are present.
- A changed source snapshot produces a distinct run identity; this is a new
  bounded run, not a claim that arbitrary historical source positions can be
  replayed.
- A final unterminated journal line is treated as an incomplete write and
  truncated on next open. A complete malformed journal entry is not repaired
  automatically.
- Transformation operations are `trim`, `lowercase`, `uppercase`, `rename`,
  `to_integer`, and explicit `replace_null`; rules must use supported `v1`
  semantics. Input records are retained separately with ordered rule
  provenance and digests.
- Quality operations are `required`, `type`, `min`, `max`, and `enum`, also
  pinned to `v1`. The metadata must choose `warning`, `reject`, or `quarantine`
  for failures. `hold` and `cleaning_required` fail closed in this profile.
  Quality never changes a record or invokes cleaning.
- Quarantined records are stored in a separate journal and are reconciled
  before a checkpoint can be written. Warning publishes the transformed
  record with quality evidence; reject fails before publication.
- Quarantine is retained with the operator-owned state directory. There is no
  automatic expiration or per-record purge command.
- `schedule --at <RFC3339>` evaluates fixed interval slots and persists
  deduplicated requests; it does not dispatch or execute them. `replay` checks
  the original plan, metadata/source digests, and pinned versions before
  reconciling the original idempotent batch. It is not a new output-producing
  replay.
- If schedule evaluation is cancelled after persisting requests, those pending
  requests receive a durable `stopped` acknowledgement and audit event without
  changing run state. `schedule-stop --state-dir <dir> --reason shutdown
  --allow-local` applies the same explicit stop policy to all pending local
  requests; a stopped slot is not returned as pending again. The local profile
  has no dispatcher, so this is not a claim that external scheduling is
  implemented.
- `clean --preview` previews an explicitly selected `v1` correction without
  persistence. Applying it writes a separate cleaned record and redacted,
  hash-only audit event; it never replaces the source or ingested record.

## Failures and recovery

Metadata, policy, source, capability, and authorization failures stop before
publication. An uncertain destination write reports an ambiguous-commit
failure; rerun the same command to reconcile the stable batch identity. Do not
delete or manually edit the state directory. A stale `LOCK` after process
termination requires an operator to verify that no process owns the directory
before removing the lock file. Preserve the full directory before any recovery.

Structured output avoids record payloads. Durable audit records correlate
activation, plan, commit, run, metadata revision, actor, and count without
record values. A separate local best-effort telemetry journal records stage
outcomes correlated by revision, plan, run, attempt, stage, and stable failure
category; it uses an allowlisted envelope and does not contain arbitrary
attributes or record values. Metrics/traces and external exporters are not yet
implemented. Audit remains required and durable even when telemetry emission
fails.

## Metadata authoring, discovery, and schedule requests

`author --metadata <draft.json> --state-dir <dir> --allow-local` stores an
immutable validated draft. `list-metadata` returns revision, status, and digest
only. Activate a stored draft by revision with `activate --revision <id>`; when
replacing an active revision, pass `--expected-active <current-id>` to prevent
a stale author from replacing a newer activation.

`discover` observes the active local JSONL object and records normalized
schema observations plus a proposal. It never mutates the active revision.
Only metadata authoring and explicit activation can make proposed changes
eligible for new runs.

## Explicit cleaning example

Given a synthetic record file `customer.json`:

```json
{"name":"  Example "}
```

Preview, then explicitly apply a correction:

On macOS or Linux:

```sh
./bin/mimir clean --record customer.json --state-dir /tmp/mimir-state \
  --classification internal --correction-id trim-name \
  --correction-version v1 --operation trim --field name \
  --preview --allow-local
./bin/mimir clean --record customer.json --state-dir /tmp/mimir-state \
  --classification internal --correction-id trim-name \
  --correction-version v1 --operation trim --field name --allow-local
```

In PowerShell on Windows:

```powershell
& .\bin\mimir.exe clean --record customer.json `
  --state-dir (Join-Path $env:TEMP 'mimir-state') `
  --classification internal --correction-id trim-name `
  --correction-version v1 --operation trim --field name `
  --preview --allow-local
& .\bin\mimir.exe clean --record customer.json `
  --state-dir (Join-Path $env:TEMP 'mimir-state') `
  --classification internal --correction-id trim-name `
  --correction-version v1 --operation trim --field name --allow-local
```

The local classification is an explicit caller-supplied ceiling, not a hosted
authorization decision. Do not use this command for confidential, restricted,
or production data.

## Compatibility and limitations

Metadata v1, activation records, plans, checkpoints, destination entries, and
audit entries are local reference contracts. Unsupported versions fail closed.
The local file format is not a supported production persistence format. The
profile is single-process per state directory; network filesystems and
multi-host ownership are unsupported. No automated backup/retention policy is
provided. The local profile does not process `confidential` or `restricted`
data.

An activated revision identifier cannot be reused for changed content. Local
state directories and journals must be private to the local user. Complete
journal entries with duplicate keys or unknown fields are rejected. During
recovery, a checkpoint without matching durable destination evidence is an
operator-recovery condition: the application will not recreate missing data
from the checkpoint alone. Preserve the full state directory and follow the
[recovery procedure](../operations/local-reference.md#recovery).

External adapters, hosted identity, service APIs, telemetry exporters,
production-grade secret stores, and production deployment remain unimplemented.
Local owner tokens and attempt history fence stale writers only within the
single-host profile; distributed worker fencing is not provided. Local rule
execution and journals are not production approved.

## Local HTTP API

The Go backend exposes the same existing local use cases for a same-origin
client. The Angular workspace provides profile identity, metadata validation
and digest-based report lookup, immutable draft authoring and explicit
activation, advisory discovery, bounded ingestion, schedule evaluation and
pending-request stopping, constrained replay, and explicit cleaning preview
and apply. These are presentation journeys over existing use cases, not new
domain behavior. The GUI does not provide run history or record queries,
schedule dispatch, proposal history, hosted identity, or production hosting.
Cleaning responses contain classified record content and must be cleared and
handled as sensitive local data. Container setup and persistence are documented in the
[three-container operations guide](../operations/local-reference.md#three-container-profile).
The versioned contract is
[`contracts/openapi/local-profile-v1.json`](../../contracts/openapi/local-profile-v1.json).
Only Linux/amd64 has been runtime-verified for this local profile.

Start the server in a terminal with an absolute, private state directory and
the explicit local grant:

```sh
mkdir -m 700 /tmp/mimir-api-state
./bin/mimir serve \
  --state-dir /tmp/mimir-api-state \
  --allow-local
```

It binds to `127.0.0.1:8787` by default. `--listen` may select another
loopback IP literal and TCP port; non-loopback addresses, hostnames, and port
zero are rejected. The state directory is fixed for the process lifetime.
Operations that read a source require a `sourceRoot` in that operation's JSON
request. Metadata and cleaning records are inline JSON; no metadata/record
filesystem path is accepted by the HTTP API. The server does not browse
directories. Stop it with Ctrl-C or SIGTERM; it stops accepting connections
and allows up to 30 seconds for in-flight requests to finish.

Every request supplies
`Mimir-Contract-Version: mimir.gui-api/v1`. The server returns the supported
contract version and an `X-Correlation-ID`; callers may supply a 1–128
character ASCII identifier using letters, digits, dot, underscore, or hyphen.
Every POST must have `Content-Type: application/json` and a matching
loopback `Origin` header. The API has no CORS support and does not accept
`OPTIONS` preflight. A browser page must be served from the same origin. The
backend forwards request cancellation and caller deadlines to the existing
use cases; it does not automatically retry operations.

For example, health is profile identity only; it is not filesystem or source
readiness:

```sh
curl -sS \
  -H 'Mimir-Contract-Version: mimir.gui-api/v1' \
  http://127.0.0.1:8787/api/v1/health
```

In the Compose profile, discovery, run, and replay source roots must be within
the read-only `/data/sources` mount. The backend's `/readyz` probe checks the
MariaDB connection separately from `/api/v1/health`; neither reports full
filesystem, source, destination, or ingestion readiness.

To validate a synthetic metadata document, include the same-origin headers on
the POST:

```sh
curl -sS \
  -H 'Mimir-Contract-Version: mimir.gui-api/v1' \
  -H 'Origin: http://127.0.0.1:8787' \
  -H 'Content-Type: application/json' \
  --data-binary @contracts/examples/metadata-v1-draft.json \
  http://127.0.0.1:8787/api/v1/metadata/validate
```

The available operations are deliberately limited:

| Method and path | Existing behavior |
| --- | --- |
| `GET /api/v1/health` | Compiled local-profile identity only |
| `POST /api/v1/metadata/validate` | Validate inline metadata; persist its safe report |
| `GET /api/v1/metadata/validation-reports/{inputDigest}` | Retrieve a report by exact digest |
| `POST /api/v1/metadata/revisions` | Author inline metadata as an immutable draft |
| `GET /api/v1/metadata/revisions` | List revision/status/digest identities only |
| `POST /api/v1/metadata/activate` | Activate exactly one inline candidate or stored draft; use `expectedActive` when replacing the active revision |
| `POST /api/v1/discovery` | Discover the active local JSONL object; supply `sourceRoot`; returns an immediate proposal only |
| `POST /api/v1/runs` | Run one bounded snapshot; supply `sourceRoot`; returns a summary, not records |
| `POST /api/v1/schedules/evaluate` | Evaluate and persist schedule requests; does not dispatch them |
| `POST /api/v1/schedules/stop` | Stop pending requests; does not cancel or alter runs |
| `POST /api/v1/replays` | Reconcile an original pinned batch; supply `sourceRoot`, `runId`, and a fixed reason |
| `POST /api/v1/cleaning/preview` | Preview an explicit correction to one inline record |
| `POST /api/v1/cleaning/apply` | Apply an explicit correction to one inline record and persist separate output/audit |

Metadata request bodies are limited to 1 MiB. An activation envelope is
limited to 1 MiB plus 4 KiB, with inline metadata still limited to 1 MiB.
Cleaning accepts one record no larger than 256 KiB in an envelope no larger
than 256 KiB plus 4 KiB. Other request bodies are limited to 64 KiB. JSON
requests reject duplicate or unknown envelope fields. The existing profile's
16 MiB/1,000-record run bounds, capability checks, classification policy,
locking, audit, and recovery semantics remain authoritative.

Errors use the `mimir.failure/v1` JSON envelope. HTTP status mapping is
`invalid_input` 400, `unsupported_capability`/`policy_rejected` 422,
`access_denied` 403, `conflict` 409, `unavailable` 503,
`ambiguous_commit`/`internal` 500, `cancelled` 408, and `timeout` 504.
The envelope's `commitCertainty`, `retryable`, and `reconciliation` values
remain authoritative; an ambiguous commit is not safe for blind retry. Raw
adapter errors are not returned. Only cleaning preview/apply responses contain
record values; handle those responses according to the request's `public` or
`internal` classification.

Unknown contract versions and unavailable history/query routes are rejected
before a use case runs. The API is versioned independently from metadata and
local journals; this backend change does not migrate persisted state. Rollback
to the prior CLI/backend binary pair must preserve the same local state
compatibility requirements. This local HTTP boundary is not safe for
shared-host or multi-user exposure and is not production approved.
