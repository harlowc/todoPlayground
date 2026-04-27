.PHONY: dev postgres-up postgres-down test test-db

dev:
	./scripts/dev

postgres-up:
	docker compose up -d postgres

postgres-down:
	docker compose down

test:
	go test ./...

test-db:
	@if [ -z "$$POSTGRES_PASSWORD" ]; then echo "POSTGRES_PASSWORD is required for make test-db"; exit 1; fi
	@set -eu; \
		export COMPOSE_PROJECT_NAME=todo-playground-test; \
		export POSTGRES_PORT="$${POSTGRES_TEST_PORT:-55432}"; \
		docker compose up -d --wait postgres; \
		trap 'docker compose down -v' EXIT; \
		RUN_DB_TESTS=1 go test ./...
