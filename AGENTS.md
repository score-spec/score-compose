# Agent instructions

## Project overview

`score-compose` is a Go implementation of the Score specification for Docker Compose.
The CLI entry point is `cmd/score-compose`; implementation packages live under
`internal/`. User-facing examples are in `examples/`.

## Development

- Follow the existing Go style and package organization. Keep changes focused and
  preserve existing behavior unless the task calls for a behavior change.
- Add or update tests alongside the code they cover. Go tests are named
  `*_test.go`; fixtures and golden files are commonly kept beside their tests.
- For changes to Score or Compose conversion behavior, check whether the examples
  or their documentation also need updating.
- Read `CONTRIBUTING.md` before preparing a contribution. In particular, Go source
  files under `cmd/` and `internal/` require the repository's Apache license header.

## Validation

Run the relevant checks for the change. The CI checks include:

```sh
go vet ./...
go test ./... -cover -race
go build ./cmd/score-compose/
```

`make test` is also documented in `CONTRIBUTING.md`; integration tests that use
Docker Compose can be skipped with `NO_DOCKER=true`.
