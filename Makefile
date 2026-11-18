GO ?= go
PYTHON ?= python3

.PHONY: test test-race contract integration lint security build package install-smoke generate-metadata-schema docs clean

test:
	$(PYTHON) scripts/check_tasks.py
	$(PYTHON) scripts/test_required_check.py
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

contract:
	$(GO) test ./...

integration:
	$(PYTHON) scripts/run_tagged.py integration

lint:
	$(PYTHON) scripts/check_format.py
	$(GO) vet ./...
	$(PYTHON) scripts/check_architecture.py
	$(PYTHON) scripts/test_architecture.py
	$(PYTHON) scripts/check_tasks.py
	$(PYTHON) scripts/check_secrets.py

security:
	$(PYTHON) scripts/check_secrets.py
	$(GO) mod verify
	$(GO) run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...

build:
	$(GO) build -trimpath -ldflags="-buildid=" -o bin/mimir ./cmd/mimir

package: build
	$(PYTHON) scripts/package.py

install-smoke:
	$(PYTHON) scripts/install_smoke.py

generate-metadata-schema:
	$(PYTHON) scripts/generate_metadata_schema.py

docs:
	$(PYTHON) scripts/check_docs.py
	$(PYTHON) scripts/check_contracts.py
	$(GO) run ./cmd/mimir validate --metadata contracts/examples/metadata-v1-draft.json
	$(PYTHON) scripts/check_tasks.py

clean:
	rm -rf bin dist
