.DEFAULT_GOAL := up

UP_FLAGS ?= --build

ifeq ($(OS),Windows_NT)
PYTHON ?= python
else
PYTHON ?= python3
endif

.PHONY: help install-prerequisites test up down

help:
	@echo Available targets:
	@echo help - Show this help
	@echo install-prerequisites - Install development and test prerequisites
	@echo test - Run all test suites
	@echo up - Build and start the stack - default
	@echo down - Stop the stack without deleting its data

install-prerequisites:
ifeq ($(OS),Windows_NT)
	powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/install-prerequisites-windows.ps1
else
	@case "$$(uname -s)" in \
		Linux) bash scripts/install-prerequisites-linux.sh ;; \
		Darwin) bash scripts/install-prerequisites-macos.sh ;; \
		*) echo "Unsupported operating system: $$(uname -s)" >&2; exit 1 ;; \
	esac
endif

test:
	$(PYTHON) scripts/run_tests.py $(UP_FLAGS)

up:
	docker compose up $(UP_FLAGS) --detach --wait --wait-timeout 180

down:
	docker compose down
