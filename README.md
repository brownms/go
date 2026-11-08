# go

Personal Go monorepo: one module (`github.com/brownms/go`), one directory per project.

| Directory | What |
|---|---|
| [`protohackers/`](protohackers/) | Solutions to the [Protohackers](https://protohackers.com) network programming challenges |
| `scratch/` | Throwaway experiments |

Code shared by two or more projects goes in a root `internal/`; until then each project keeps its own.

Tool versions and tasks live in `.mise.toml`. Run `mise install` once, then `mise tasks` to list what's available:

| Task | Does |
|---|---|
| `mise run build` | Build every command into `out/` |
| `mise run test` | `go test -race ./...` |
| `mise run lint` | golangci-lint (linters and formatter checks) |
| `mise run fmt` | Apply gofmt and goimports |
| `mise run ci` | Lint and test, as GitHub Actions runs it |
