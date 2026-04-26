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

## Test

```sh
go test ./...
```

If the Go build cache is not writable in your environment, use a local or temporary cache:

```sh
GOCACHE=/tmp/test1-go-cache go test ./...
```

## Notes

Todos are stored in memory. Restarting the server clears the list.
