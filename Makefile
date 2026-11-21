.DEFAULT_GOAL := help

ifeq ($(OS),Windows_NT)
PYTHON ?= py -3.12
BINARY ?= bin/mimir.exe
else
PYTHON ?= python3
BINARY ?= bin/mimir
endif

GO ?= $(shell $(PYTHON) scripts/setup_dev.py --print-command)
GOFMT ?= $(shell $(PYTHON) scripts/setup_dev.py --print-gofmt)

.PHONY: help run test test-race contract integration lint security build package install-smoke generate-metadata-schema docs clean

help:
	@echo Available targets:
	@echo   run                     Start the local stack and open the frontend
	@echo   test                    Run repository checks and Go tests
	@echo   test-race               Run Go tests with the race detector
	@echo   contract                Run Go contract tests
	@echo   integration             Run integration tests
	@echo   lint                    Run formatting, architecture, and security checks
	@echo   security                Run security checks
	@echo   build                   Build the Mimir executable
	@echo   package                 Build a distributable package
	@echo   install-smoke           Run the installation smoke test
	@echo   generate-metadata-schema Generate the metadata schema
	@echo   docs                    Validate documentation and contracts
	@echo   clean                   Remove build artifacts

run:
	docker compose up --build --wait
	$(PYTHON) -c "import subprocess, sys, webbrowser; address = subprocess.check_output(['docker', 'compose', 'port', 'frontend', '80'], text=True).strip(); address or sys.exit('Could not determine frontend address'); url = 'http://' + address; print('Opening ' + url, flush=True); webbrowser.open(url) or sys.exit('Could not open browser; open ' + url + ' manually')"

test:
	$(PYTHON) scripts/check_tasks.py
	$(PYTHON) scripts/test_required_check.py
	$(PYTHON) scripts/test_setup_dev.py
	$(PYTHON) scripts/test_run_race_tests.py
	$(PYTHON) scripts/test_artifact_identity.py
	$(PYTHON) scripts/test_tool_pins.py
	$(GO) test ./...

test-race:
	$(PYTHON) scripts/run_race_tests.py --go "$(GO)"

contract:
	$(GO) test ./...

integration:
	$(PYTHON) scripts/run_tagged.py integration

lint:
	$(PYTHON) scripts/check_format.py --gofmt "$(GOFMT)"
	$(GO) vet ./...
	$(PYTHON) scripts/check_architecture.py
	$(PYTHON) scripts/test_architecture.py
	$(PYTHON) scripts/check_tasks.py
	$(PYTHON) scripts/check_secrets.py

security:
	$(PYTHON) scripts/check_secrets.py
	$(GO) mod verify
	$(GO) run golang.org/x/vuln/cmd/govulncheck@v1.10.0 ./...

build:
	$(GO) build -trimpath -ldflags="-buildid=" -o $(BINARY) ./cmd/mimir

package: build
	$(PYTHON) scripts/package.py

install-smoke:
	$(PYTHON) scripts/install_smoke.py --go "$(GO)"

generate-metadata-schema:
	$(PYTHON) scripts/generate_metadata_schema.py

docs:
	$(PYTHON) scripts/check_docs.py
	$(PYTHON) scripts/check_contracts.py
	$(PYTHON) scripts/check_openapi.py
	$(GO) run ./cmd/mimir validate --metadata contracts/examples/metadata-v1-draft.json
	$(PYTHON) scripts/check_tasks.py

clean:
	rm -rf bin dist
