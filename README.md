# todoPlayground

A small Go and HTMX todo playground. It renders a server-side HTML page, then uses HTMX requests to add, edit, update, cancel, and remove todos without a full page refresh.

## Requirements

- Go 1.26.2 or newer

## Run

```sh
go run .
```

Then open:

```text
http://localhost:8080
```

To start the app and Dockerized Postgres together for local development:

```sh
make dev
```

This starts the `postgres` service, waits for it to become healthy, then runs the Go app with `TODO_STORE=postgres`.
Press `Ctrl-C` to stop the Go app. Run `make postgres-down` when you want to stop the Postgres container too.

The app can read its server and Postgres settings from environment variables:

- `SERVER_ADDR` default: `:8080`
- `TODO_STORE` default: `memory`
- `POSTGRES_HOST` default: `localhost`
- `POSTGRES_PORT` default: `5432`
- `POSTGRES_DB` default: `todo_playground`
- `POSTGRES_USER` default: `todo_user`
- `POSTGRES_PASSWORD` required when `TODO_STORE=postgres`
- `POSTGRES_SSLMODE` default: `disable`

## Local Postgres

To start only the local Postgres container:

```sh
make postgres-up
```

To stop it and remove the Compose network:

```sh
make postgres-down
```

The container exposes Postgres on `localhost:5432` with:

- database: `todo_playground`
- user: `todo_user`
- password: whatever you set in your local `.env` file or shell as `POSTGRES_PASSWORD`

By default the app still uses in-memory storage. To run it against Postgres:

```sh
TODO_STORE=postgres go run .
```

## Example Env File

You can keep local settings in a `.env` file. That file is ignored by Git. Start from the committed template, then choose your own local password:

```sh
cp .env.example .env
```

A local setup should look like:

```dotenv
SERVER_ADDR=:8080
TODO_STORE=postgres
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=todo_playground
POSTGRES_USER=todo_user
POSTGRES_PASSWORD=replace-with-a-local-secret
POSTGRES_SSLMODE=disable
```

## Postgres Check

To verify the app can reach Postgres and apply the initial schema migration:

```sh
make test-db
```

If the Go build cache is not writable in your environment, use:

```sh
RUN_DB_TESTS=1 GOCACHE=/tmp/test1-go-cache go test ./...
```

## Test

```sh
make test
```

If the Go build cache is not writable in your environment, use a local or temporary cache:

```sh
GOCACHE=/tmp/test1-go-cache go test ./...
```

## Notes

Use `TODO_STORE=memory` for ephemeral local state or `TODO_STORE=postgres` for Dockerized Postgres-backed persistence.
HTMX is vendored in `static/htmx.min.js` so the app does not need to load it from a CDN at runtime.
