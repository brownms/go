# go

Personal Go monorepo: one module (`github.com/brownms/go`), one directory per project.

| Directory | What |
|---|---|
| [`protohackers/`](protohackers/) | Solutions to the [Protohackers](https://protohackers.com) network programming challenges |
| `scratch/` | Throwaway experiments |

Code shared by two or more projects goes in a root `internal/`; until then each project keeps its own.

Tool versions are pinned in `.mise.toml`; run `mise install` to get them.
