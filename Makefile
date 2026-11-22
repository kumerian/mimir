.DEFAULT_GOAL := up

UP_FLAGS ?= --build

.PHONY: test up down

test:
	cd backend && ./mvnw --batch-mode --no-transfer-progress verify
	cd frontend && npm ci && npm test
	@set -eu; \
	export COMPOSE_PROJECT_NAME="mimir-test-$$$$" FRONTEND_PORT=0; \
	trap 'docker compose down --volumes --remove-orphans' EXIT; \
	docker compose up $(UP_FLAGS) --detach --wait --wait-timeout 180; \
	export STACK_URL="http://$$(docker compose port frontend 80)"; \
	python3 -m unittest discover -s tests/docker -v

up:
	docker compose up $(UP_FLAGS) --detach --wait --wait-timeout 180

down:
	docker compose down
