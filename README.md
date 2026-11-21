# Mimir

Mimir is a platformless data-ingestion toolset. It aims to keep ingestion rules,
data semantics, scheduling, validation, and cleaning independent of any
specific database, cloud platform, orchestration engine, or business domain.
Adapters can connect the core model and execution contracts to concrete
technologies.

## Goals

Mimir will:

- discover and maintain metadata for data sources, objects, and columns;
- schedule ingestion from metadata rather than tool-specific configuration;
- load source data through interchangeable source and destination adapters;
- normalize data into consistent representations;
- apply functional/business mappings to source-agnostic data;
- validate, profile, and optionally clean ingested data;
- record execution state and data-quality outcomes for audit and recovery.

The repository has a working Go foundation and a deliberately limited local
reference ingestion path. The broader toolset below remains intended product
behavior unless explicitly listed as implemented in
[`docs/user/local-reference-ingestion.md`](docs/user/local-reference-ingestion.md).

## Getting started

The local developer prerequisites are Go 1.27.4, GNU Make, Git, and Python
3.12. Follow the [local development setup guide](docs/operations/local-development.md)
to install them and configure your shell. Then run the hermetic checks with:

```sh
make test
make lint
make docs
make build
make install-smoke
```

`make install-smoke` builds into a temporary clean install directory and checks
health, metadata validation/report lookup, activation, and a synthetic local
run; it removes the temporary install and data on exit. It verifies the current
host only. CI also runs this smoke on its native Linux, macOS, and Windows
runners, while the six-target matrix cross-builds all declared OS/architecture
combinations.

See the [local reference ingestion guide](docs/user/local-reference-ingestion.md)
for metadata validation, explicit local activation, a synthetic bounded run,
supported guarantees, and recovery. The [operations guide](docs/operations/local-reference.md)
describes local state ownership and recovery. The file-based reference profile
is not suitable for production.

The Go backend also exposes the existing local-profile operations through a
loopback-only HTTP/JSON API. Start it with an absolute private state directory
and explicit local grant, for example
`./bin/mimir serve --state-dir /absolute/path/to/mimir-state --allow-local`.
The default listener is `127.0.0.1:8787`. The Angular screens and production
journeys remain in progress; the container profile includes an Angular
connectivity shell. See the API and local-server instructions in the
[local reference guide](docs/user/local-reference-ingestion.md#local-http-api).

## Local containers

Docker Compose starts the Angular/Nginx frontend, Go backend, and MariaDB
metadata store. The browser reaches the Go API through the same-origin Nginx
proxy; only the frontend is published, on host loopback port 8080. Metadata
revisions, validation reports, and discovery proposals use the MariaDB
`metadata` schema. Execution state and data remain on a separate backend
volume. This remains a single-user local profile, not a production or
remote-access deployment.

Create local-only credentials, then start the services:

```sh
printf 'MARIADB_ROOT_PASSWORD=%s\nMIMIR_DB_RUNTIME_PASSWORD=%s\nMIMIR_DB_MIGRATION_PASSWORD=%s\n' \
  "$(openssl rand -hex 32)" "$(openssl rand -hex 32)" "$(openssl rand -hex 32)" > .env
chmod 600 .env
make run
```

With Docker Compose v2, `make run` builds and starts the services, waits for
them to become healthy, and opens the frontend in the default browser. It
requires GNU Make and Python 3.12 on Windows (or Python 3 on other platforms).
If the browser cannot be opened, use the displayed URL manually. The default
URL is `http://127.0.0.1:8080`; `MIMIR_HTTP_PORT` can change the host port.
Add permitted local JSONL sources under `data/sources`. The backend accepts source roots only
within that read-only mount. To stop without deleting persistent data, run
`docker compose down`; `docker compose down --volumes` permanently deletes
both named volumes. See the [container operations guide](docs/operations/local-reference.md#three-container-profile)
for readiness, backup, restore, and credential handling.

## Product intent

### Metadata Management

The metadata service is the control plane for the other tools. It retrieves,
stores, validates, and updates a model for:

- **Technical metadata** — source type, connectivity contract, object type,
  column types, constraints, keys, partitions, and incremental-load fields.
- **Functional metadata** — business meaning, ownership, classification,
  sensitivity, mappings, canonical definitions, and domain terminology.
- **Scheduling metadata** — cadence, dependencies, activation windows, retry
  policy, concurrency, load strategy, scope, and per-source/object/column
  overrides.
- **Lineage and governance metadata** — versions, provenance, transformations,
  relationships, and effective dates.

Metadata must be retrievable independently of discovery and storage
implementations. Changes should be versioned, validated before activation, and
auditable.

### Ingestion

The ingestion service is the data-movement execution plane. It reads active
metadata to:

1. determine eligible sources, objects, and columns;
2. create or resume an ingestion run;
3. choose the required source and destination adapters;
4. apply a full, incremental, streaming, or replay strategy;
5. preserve required raw values and load provenance;
6. persist checkpoints, metrics, failures, and completion state.

Scheduling policies may be evaluated by an embedded scheduler or exported to
an external orchestrator. Runs should be idempotent, recoverable, and observable.

### Transformation

The transformation service converts source-specific structures and technical
representations into a source-agnostic model, then applies functional metadata
to map the data to canonical business concepts.

Examples include type normalization, null handling, unit conversion, naming
standardization, key mapping, and enrichment. Transformations should be
explicit, ordered, versioned, and reproducible. Input, output, metadata
version, and transformation failures should be traceable for every run.

### Data Quality and Cleaning

The quality service runs after ingestion, or at a configured pipeline boundary,
and uses metadata-defined rules to:

- validate schemas, types, ranges, formats, uniqueness, and referential
  integrity;
- detect missing, stale, duplicate, or anomalous values;
- compare outcomes with source and historical baselines;
- quarantine or mark invalid records;
- optionally apply approved, deterministic cleaning rules.

Automated cleaning must be explicit and auditable. Unless metadata requires
otherwise, validation should report or quarantine data rather than silently
alter it.

## Conceptual Architecture

```text
Metadata discovery / authoring
              |
              v
      Metadata repository ----> Scheduler / orchestrator
              |                         |
              |                         v
              +----> Ingestion / transformation runtime
                         |                |
Source adapters ---------+                +--------> Destination adapters
                                          |
                                          v
                              Quality and cleaning
                                          |
                              Metrics, lineage, audit, state
```

The architecture should separate:

- **domain contracts** from platform-specific adapters;
- **metadata and policy** from run-time implementation;
- **orchestration** from ingestion and transformation logic;
- **execution state** from the ingested data itself;
- **quality detection** from mutation/cleaning.

## Core Domain Model

At a minimum, the metadata model should represent:

| Entity | Purpose |
| --- | --- |
| Source | Connection and capability definition |
| Object | Table, file, endpoint, topic, or other source unit |
| Column | Atomic field, including type and technical constraints |
| Mapping | Technical-to-canonical field or value conversion |
| Schedule | Trigger, scope, dependencies, and retry behavior |
| Rule | Validation, cleaning, transformation, or policy |
| Run | Immutable execution attempt and its checkpoints |
| Lineage Record | Input, metadata version, operation, and output relationship |

Relationships and overrides should flow from source to object to column, with
more specific configuration overriding broader defaults.

## Execution Flow

1. Discover or author metadata.
2. Validate and activate a versioned metadata set.
3. Evaluate schedules, dependencies, and capabilities.
4. Plan a run and resolve the applicable adapters and policies.
5. Ingest raw data and record checkpoints and provenance.
6. Normalize and transform data using the pinned metadata version.
7. Run quality rules and optionally clean or quarantine records.
8. Publish the result and record metrics, lineage, and final run state.

Failures should retain enough state to retry from a safe checkpoint without
duplicating committed data.

## Extension Points

Platform neutrality is achieved through stable contracts for, at minimum:

- metadata repositories;
- source and destination adapters;
- metadata discovery providers;
- schedulers and orchestrators;
- transformation engines;
- quality-rule engines;
- state, checkpoint, metric, and audit stores.

Adapters should report capabilities and reject unsupported plans explicitly.
The core must not depend on the APIs of a particular storage, compute, cloud,
messaging, or workflow platform.

## Architectural Requirements

An initial implementation should support:

- metadata and pipeline versioning;
- configurable batch and event-driven execution;
- idempotent or exactly-once destination behavior where supported;
- deterministic replays and repeatable transformations;
- schema drift detection and controlled metadata updates;
- secrets referenced rather than stored in metadata;
- structured logs, metrics, traces, and failure reasons;
- least-privilege adapter access and data-classification controls;
- source, object, column, and rule-level access control;
- contract tests that allow interchangeable adapters.

## Suggested Repository Evolution

1. Define core domain types, metadata formats, and adapter contracts.
2. Implement metadata validation, versioning, and persistence.
3. Add a minimal in-process scheduler and reference adapters.
4. Implement run planning, checkpoints, idempotent ingestion, and replay.
5. Add transformations with pinned metadata versions.
6. Add quality rules, quarantine, and opt-in cleaning.
7. Integrate external stores, orchestrators, and observability platforms.

## Non-Goals

Mimir is not tied to a prescribed database, cloud, workflow engine, messaging
system, data format, or transformation language. The project should also avoid
silently correcting data or migrating metadata versions without an explicit,
reviewable policy.
