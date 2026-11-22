# Technical setup

## Architecture

The scaffold runs exactly three containers:

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

This is an infrastructure scaffold, not an ingestion implementation. There are
no domain tables, migrations, authentication, or ingestion workflows yet.

## Repository layout

- `backend/`: Spring Boot application, Maven wrapper, Java tests, Dockerfile.
- `frontend/`: Angular application, Vitest tests, lockfile, development proxy,
  Nginx configuration, Dockerfile.
- `docker/mariadb/`: buildable image based on the official MariaDB image.
- `tests/docker/`: Python standard-library integration tests.
- `Makefile`: shortcuts for running all tests and the local containers.
- `.github/workflows/ci.yml`: application tests, stack tests, then image builds.
- `docs/`: project documentation.

## Run the complete stack

Prerequisites: GNU Make and Docker Engine with Docker Compose v2. Python 3 is required to run
the container integration tests. Host Java and Node are not needed for Docker.

```sh
cp .env.example .env
make up
```

Open `http://localhost:8080`. The page reports database connectivity through the
backend. Inspect failures with `docker compose logs --no-color`.

The example credentials are **local-development only**. Change both passwords
in `.env` before using shared infrastructure; `.env` is ignored by Git and
credentials are supplied at runtime, never baked into images. A non-root
database account is used by Java. The backend image also runs as a non-root user.
For production, supply managed secrets, TLS, access controls, and an appropriate
external reverse proxy rather than exposing this scaffold directly.

`DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_ROOT_PASSWORD`, and `FRONTEND_PORT` are
configurable in `.env`. Database passwords are required. `IMAGE_TAG` defaults to
`local`. MariaDB initialization settings only apply to an empty database volume;
changing `.env` does not rotate credentials in an existing database.

```sh
make down
```

This stops the stack while retaining its database. To intentionally erase the
project's database, use `docker compose down --volumes`. Do not run that command
against a stack whose data you need to keep.

## Local application development

Prerequisites: JDK 21 and Node.js 24 with npm. Maven is downloaded by the checked-in
wrapper on first use. Angular dependencies are pinned in `package-lock.json`.

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
all suites from the repository root:

```sh
cp .env.example .env # First use only; do not overwrite existing credentials.
make test
```

`make test` runs Java verification, installs the locked frontend dependencies and
runs Angular tests, then builds and tests all three containers. Docker tests use
a unique Compose project and a dynamically assigned frontend port. Their
containers and temporary database volume are cleaned up even if startup or
tests fail; the normal local stack and its database are left untouched.
`make up` builds and starts the local stack, waiting for healthy containers.
`make down` stops it without deleting database storage. Plain `make` defaults to
`make up`.

The equivalent individual suite commands are:

```sh
(cd backend && ./mvnw --batch-mode --no-transfer-progress verify)
(cd frontend && npm ci && npm test)
docker compose up --build --detach --wait --wait-timeout 180
python3 -m unittest discover -s tests/docker -v
docker compose down
```

Java tests cover status success/failure and Spring HTTP/health integration using
an isolated H2 database. Angular tests cover loading, the successful API response,
and visible request errors without a live backend. Docker tests verify exactly
three healthy containers, Angular assets, SPA navigation, real MariaDB-backed API
access through Nginx, and correct missing-asset/API errors. They complement,
rather than replace, the H2 tests.

Set `STACK_URL` when testing a non-default frontend port. Use a unique
`COMPOSE_PROJECT_NAME` and `FRONTEND_PORT` for independent stacks on a shared host.
The test process must use the same Compose project/environment as startup.

## CI and build artifacts

GitHub Actions runs on pull requests, pushes to `main`, and manual dispatch.
Java and Angular tests run independently. When both pass, the Docker test job
builds temporary `:test` images and starts all three containers for integration
checks, emitting logs on failure and always cleaning up its resources.

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
