# go

Personal Go monorepo: one module (`github.com/brownms/go`), one directory per project.

| Directory | What |
|---|---|
| [`protohackers/`](protohackers/) | Solutions to the [Protohackers](https://protohackers.com) network programming challenges |
| `scratch/` | Throwaway experiments; `go run ./scratch` |

## Adding a project

Create a top-level directory with a `cmd/<command>/main.go` for each binary and the rest of its code under `internal/`. Lint, test, build and CI find it automatically; nothing needs registering. Code shared by two or more projects goes in a root `internal/`; until then each project keeps its own.

## Tasks

Tool versions and tasks live in `.mise.toml`. Run `mise install` once, then `mise tasks` to list what's available:

| Task | Does |
|---|---|
| `mise run build` | Build every `<project>/cmd/<command>` into `out/<project>/` |
| `mise run test` | `go test -race ./...` |
| `mise run lint` | golangci-lint (linters and formatter checks) |
| `mise run fmt` | Apply gofmt and goimports |
| `mise run ci` | Lint, test and build, as GitHub Actions runs it |
| `mise run run <project>/<command> -- <args>` | `go run ./<project>/cmd/<command> <args>` |
| `mise run publish <project>/<command>` | Cross-compile one command for linux/amd64 and scp it to host `yoda` |
