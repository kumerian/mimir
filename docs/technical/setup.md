# Technical setup

## Architecture

The application runs exactly three containers:

| Service | Stack | Connectivity |
| --- | --- | --- |
| `frontend` | Angular 21, built with Node 24, served by Nginx | Published at `http://localhost:8080`; proxies `/api/` to `backend:8080` |
| `backend` | Java 21, Spring Boot 3.5, Maven 3.9 | JDBC to `database:3306`; HTTP on internal port 8080 |
| `database` | MariaDB 11.4 LTS | Internal port 3306; persistent named volume |

The browser uses relative `/api` URLs, so Docker-only DNS names never reach
the browser and CORS configuration is unnecessary. Compose creates the internal
network. Neither the backend nor database publishes a host port. The frontend
port is bound to loopback by default.

`GET /api/status` executes `SELECT 1` against the database and returns
`{"application":"mimir","database":"connected"}` only on success. Database
errors propagate as HTTP errors rather than a false success response.
`GET /actuator/health` is internal and includes Spring's database health check.
Nginx's `/healthz` checks the frontend server. Compose waits for MariaDB health
before starting Java, and Java health before starting Nginx.

The [ingestion metadata model](ingestion-metadata.md) is implemented with
Liquibase migrations, Spring Data JDBC records, CRUD REST endpoints, and an
Angular editor. There is no login/authentication layer or ingestion execution
workflow. Authentication credentials and API parameter values are stored
unencrypted as supplied and returned by the unauthenticated API, so only use
trusted local-development values, not production secrets.

See [logging and metadata audit](logging.md) for log-level configuration,
database retention, and the web audit history.

## Repository layout

- `backend/`: Spring Boot application, Maven wrapper, Java tests, Dockerfile.
- `frontend/`: Angular application, Vitest tests, lockfile, development proxy,
  Nginx configuration, Dockerfile.
- `docker/mariadb/`: buildable image based on the official MariaDB image.
- `tests/docker/`: Python standard-library integration tests.
- `scripts/`: test runner and platform-specific prerequisite installers.
- `Makefile`: shortcuts for installing prerequisites, running tests, and local containers.
- `.github/workflows/ci.yml`: application tests, stack tests, then image builds.
- `docs/`: project documentation.

Run `make help` to list the available Make targets.

## Install prerequisites

The installers set up GNU Make, JDK 21, Node.js 24 with npm, Python 3, and Docker
with Compose v2 for development and tests. They install system packages and may
prompt for administrator access or package agreements. Review the scripts before
running them, especially on a shared machine.

Install the platform package manager first: [Windows Package Manager](https://learn.microsoft.com/windows/package-manager/winget/)
on Windows or [Homebrew](https://brew.sh/) on macOS. On Linux, the installer
supports Ubuntu, Debian, and Fedora releases and architectures supported by the
Docker, NodeSource, and JDK package repositories; it requires root or sudo.
It adds Docker and NodeSource repositories, plus Adoptium on Debian/Ubuntu.
Existing conflicting packages or repository configuration may need manual
resolution; the installer does not remove them.

If Make is already available, run `make install-prerequisites`. Otherwise,
bootstrap directly from the repository root:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/install-prerequisites-windows.ps1
```

```sh
# macOS
bash scripts/install-prerequisites-macos.sh
# Ubuntu, Debian, or Fedora
bash scripts/install-prerequisites-linux.sh
```

Open a new terminal after installation. Set `JAVA_HOME` to the JDK 21 directory;
on macOS, follow the installer's printed `JAVA_HOME` and `PATH` instructions
to select the keg-only Node.js 24 and JDK. On Windows, ensure the installed
tools are on `PATH`. Start Docker Desktop and complete its first-run setup on
Windows or macOS. On Linux, start the Docker service and configure access using
[Docker's post-installation instructions](https://docs.docker.com/engine/install/linux-postinstall/).
Membership in the Docker group grants root-equivalent privileges; the installer
does not add users to it. Maven is provided by the checked-in wrapper and
frontend dependencies are installed by `npm ci`.

## Run the complete stack

The supported hosts are Windows, macOS, and Linux. Install GNU Make and Docker
Engine with Docker Compose v2 (Docker Desktop can provide Docker on Windows and
macOS). Host Java and Node are not needed to run the Docker stack.

```sh
cp .env.example .env
make up
```

On Windows, run `Copy-Item .env.example .env` in PowerShell instead of `cp`.
These commands copy the example credentials only for first-time setup; do not
overwrite an existing `.env`.

`make run` is an alias for `make up` and starts the entire suite of application
services. Open `http://localhost:8080`. The page reports database connectivity and
provides the [metadata editor](../user/ingestion-metadata.md). Inspect failures
with `docker compose logs --no-color`. Liquibase upgrades the database at backend
startup; retain existing volumes when deploying new application versions.

The example credentials are **local-development only**. Change both passwords
in `.env` before using shared infrastructure; `.env` is ignored by Git and
credentials are supplied at runtime, never baked into images. A non-root
database account is used by Java. The backend image also runs as a non-root user.
For production, supply managed secrets, TLS, access controls, and an appropriate
external reverse proxy rather than exposing this scaffold directly.

`DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_ROOT_PASSWORD`, `FRONTEND_PORT`, and
`LOG_LEVEL` are configurable in `.env`. Database passwords are required.
`LOG_LEVEL` defaults to `INFO`; see [logging and metadata audit](logging.md)
for supported values. `IMAGE_TAG` defaults to `local`. MariaDB initialization
settings only apply to an empty database volume; changing `.env` does not rotate
credentials in an existing database.

```sh
make down
```

This stops the stack while retaining its database. To intentionally erase the
project's database, use `docker compose down --volumes`. Do not run that command
against a stack whose data you need to keep.

## Local application development

Prerequisites: JDK 21 and Node.js 24 with npm. Set `JAVA_HOME` to the JDK
directory, especially on Windows where the Maven wrapper requires it. Maven is
downloaded by the checked-in wrapper on first use. Angular dependencies are pinned
in `package-lock.json`.

For Java development, start MariaDB with a development-only port override.
Create a local file outside version control, such as `/tmp/mimir-dev-compose.yaml`:

```yaml
services:
  database:
    ports:
      - "127.0.0.1:3306:3306"
```

```sh
docker compose -f compose.yaml -f /tmp/mimir-dev-compose.yaml up --build --detach --wait database
cd backend
DB_PASSWORD=local-development-password ./mvnw spring-boot:run
```

Use the same password as `.env`. The backend defaults to the local `mimir`
database with user `mimir`; override `DB_URL` and `DB_USER` as needed.
On Windows, create the override file at a local path and use that path in the
Compose command. In PowerShell, start the backend with
`$env:DB_PASSWORD = "local-development-password"; .\mvnw.cmd spring-boot:run`
from the `backend` directory.
In another terminal:

```sh
cd frontend
npm ci
npm start
```

Visit `http://localhost:4200`. The Angular development server proxies `/api/**`
to the host backend at `http://localhost:8080`. Stop the containerized frontend
first if it already occupies that port.

## Test suites

With GNU Make, JDK 21, Node.js 24, Python 3, and Docker Compose installed, run
all suites from the repository root. Start Docker first and set `JAVA_HOME` to
the JDK directory:

```sh
make test
```

`make test` runs Java verification, installs the locked frontend dependencies and
runs Angular tests, then builds and tests all three containers. Docker tests use
a unique Compose project, generated database passwords, and a dynamically
assigned frontend port. No `.env` file is required for `make test`; generated
passwords override local database passwords only in the test subprocesses. Their
containers and temporary database volume are cleaned up even if startup or
tests fail; Compose logs are printed on failure. The normal local stack and its
database are left untouched.
`make up` builds and starts the local stack, waiting for healthy containers.
`make run` does the same; `make test` runs all test suites rather than starting
a persistent application stack.
`make down` stops it without deleting database storage. Plain `make` defaults to
`make up`.

If npm is not on `PATH` but [fnm](https://github.com/Schniz/fnm) is installed,
the runner executes frontend commands through `fnm exec --using 24`. Install
that runtime once with `fnm install 24`; initializing fnm in your shell is not
required for `make test`. When npm is already on `PATH`, it is used as before.

On Windows, use PowerShell or Command Prompt with GNU Make on `PATH`. The test target
uses `python` on Windows and `python3` on macOS and Linux. Override it if needed,
for example `make test PYTHON=python3`. The Windows Maven wrapper needs
`JAVA_HOME`, not just `java` on `PATH`.

The individual suite commands on macOS and Linux are:

```sh
(cd backend && ./mvnw --batch-mode --no-transfer-progress verify)
(cd frontend && npm ci && npm test)
docker compose up --build --detach --wait --wait-timeout 180
python3 -m unittest discover -s tests/docker -v
docker compose down
```

On Windows, use `.\mvnw.cmd --batch-mode --no-transfer-progress verify` from
`backend`, and `python -m unittest discover -s tests/docker -v` for the Docker
tests. The Docker commands require the same Compose project and environment
throughout; unlike `make test`, the individual commands above use the local
stack and do not create or clean up an isolated test project.

Java tests cover status success/failure, HTTP/health integration, and the complete
Liquibase-backed metadata API using isolated H2 databases. Angular tests cover
loading, CRUD forms, relationship choices, validation, and visible request errors
without a live backend. CI also runs a production Angular build on each host.
Docker tests verify metadata CRUD, referential integrity, persistence and migration
idempotence after a backend restart, and exactly
three healthy containers, Angular assets, SPA navigation, real MariaDB-backed API
access through Nginx, and correct missing-asset/API errors. They complement,
rather than replace, the H2 tests.

Set `STACK_URL` when testing a non-default frontend port. Use a unique
`COMPOSE_PROJECT_NAME` and `FRONTEND_PORT` for independent stacks on a shared host.
The test process must use the same Compose project/environment as startup.

## CI and build artifacts

GitHub Actions runs on pull requests, pushes to `main`, and manual dispatch.
Java, Angular, and test-runner checks run on Windows, macOS, and Linux. When
they pass, the Linux Docker test job runs `make test` end to end, building
temporary `:test` images and starting all three containers for integration
checks.

Only after **all** tests pass does the final build job build the three images
tagged with the commit SHA. It exports them as a compressed Docker image archive
in a GitHub Actions artifact, retained for seven days. There is no registry push
or automatic deployment. Test-image builds necessarily precede container tests;
the final deliverable build is gated on the complete test suite.

Load a downloaded archive and run the same wiring without rebuilding:

```sh
gunzip -c mimir-images.tar.gz | docker load
IMAGE_TAG=<commit-sha> docker compose up --no-build --detach --wait
```

Supply `.env` on the deployment host. The archive includes all three images;
database storage and credentials are intentionally not included. The MariaDB
Dockerfile inherits the official image unchanged, including its initialization
and healthcheck tooling.

## Builds behind a private certificate authority

On hosts that intercept HTTPS using a private CA, Java may reject dependency
downloads even when host tools work. Supply a trusted Java certificate store as
the optional BuildKit secret; do not disable TLS verification:

```sh
docker build --secret id=maven-cacerts,src=/path/to/trusted/cacerts \
  -t mimir-backend:local backend
docker compose build frontend database
docker compose up --no-build --detach --wait
```

The secret is mounted only during Maven build steps, is not copied into image
layers, and is not needed on hosts using the standard public trust chain.
After building trusted images as above, use `make up UP_FLAGS=--no-build` or
`make test UP_FLAGS=--no-build` to reuse them without triggering another build.
