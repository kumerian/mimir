.DEFAULT_GOAL := up

UP_FLAGS ?= --build

ifeq ($(OS),Windows_NT)
PYTHON ?= python
else
PYTHON ?= python3
endif

.PHONY: test up down

test:
	$(PYTHON) scripts/run_tests.py $(UP_FLAGS)

up:
	docker compose up $(UP_FLAGS) --detach --wait --wait-timeout 180

down:
	docker compose down
