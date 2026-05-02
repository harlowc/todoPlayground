# todoPlayground

A small Go and HTMX todo playground. It renders a server-side HTML page, then uses HTMX requests to add, edit, update, cancel, complete, archive, recreate for tomorrow, and remove todos without a full page refresh. Todos can also have optional due dates, categories, priorities, notes, search, and simple views for all, active, completed, scheduled, today, and upcoming tasks.

## Requirements

- Go 1.26.2 or newer
- Docker with Docker Compose for local Postgres

## Run

To start the app and Dockerized Postgres together for local development:

```sh
make dev
```

Then open:

```text
http://localhost:8080
```

This starts the `postgres` service, waits for it to become healthy, then runs the Go app against Postgres.
Press `Ctrl-C` to stop the Go app. Run `make postgres-down` when you want to stop the Postgres container too.

The app can read its server and Postgres settings from environment variables:

- `SERVER_ADDR` default: `:8080`
- `POSTGRES_HOST` default: `localhost`
- `POSTGRES_PORT` default: `5432`
- `POSTGRES_DB` default: `todo_playground`
- `POSTGRES_USER` default: `todo_user`
- `POSTGRES_PASSWORD` required
- `POSTGRES_SSLMODE` default: `disable`

## Local Postgres

To start only the local Postgres container:

```sh
POSTGRES_PASSWORD=replace-with-a-local-secret make postgres-up
```

If you use `make dev`, `scripts/dev` loads `.env` before starting Docker Compose, so you can keep the password there for normal local development.

To stop it and remove the Compose network:

```sh
make postgres-down
```

The container exposes Postgres on `localhost:5432` with:

- database: `todo_playground`
- user: `todo_user`
- password: whatever you set in your local `.env` file or shell as `POSTGRES_PASSWORD`

To run the app manually against Postgres:

```sh
POSTGRES_PASSWORD=replace-with-a-local-secret go run .
```

## Example Env File

You can keep local settings in a `.env` file. That file is ignored by Git.

A local setup should look like:

```dotenv
SERVER_ADDR=:8080
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=todo_playground
POSTGRES_USER=todo_user
POSTGRES_PASSWORD=replace-with-a-local-secret
POSTGRES_SSLMODE=disable
```

## Postgres Check

To verify the app can reach Postgres and apply all schema migrations:

```sh
POSTGRES_PASSWORD=replace-with-a-local-secret make test-db
```

`make test-db` starts an isolated Postgres container on `POSTGRES_TEST_PORT`, defaulting to `55432`, runs the full Go test suite with `RUN_DB_TESTS=1`, then removes the test database volume.

If you already have a Postgres database running with the expected schema settings, you can run the DB tests directly:

```sh
RUN_DB_TESTS=1 POSTGRES_PASSWORD=replace-with-a-local-secret go test ./...
```

If the Go build cache is not writable in your environment, add a local or temporary cache:

```sh
RUN_DB_TESTS=1 POSTGRES_PASSWORD=replace-with-a-local-secret GOCACHE=/tmp/test1-go-cache go test ./...
```

## Test

```sh
make test
```

`make test` runs the fast non-DB suite. It skips the Postgres integration tests unless `RUN_DB_TESTS=1` is set.

If the Go build cache is not writable in your environment, use a local or temporary cache:

```sh
GOCACHE=/tmp/test1-go-cache go test ./...
```

## Current Features

- Add, edit, complete, archive, remove, and recreate todos.
- Persist due dates, categories, priorities, and notes in Postgres.
- View all, active, completed, scheduled, today, and upcoming todos.
- Filter by category and priority.
- Search todo text, category, and notes.
- Hide archived completed todos from the normal views.
- Offer the next weekday when recreating a task would otherwise land on a weekend.

## Notes

The app is intended to run with Dockerized Postgres-backed persistence.
HTMX is vendored in `static/htmx.min.js` so the app does not need to load it from a CDN at runtime.
