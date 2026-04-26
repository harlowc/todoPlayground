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
	docker compose up -d postgres
	RUN_DB_TESTS=1 go test ./...
