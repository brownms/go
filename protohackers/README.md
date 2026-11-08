# protohackers

Solutions to [protohackers.com](https://protohackers.com) challenges, written while learning Go.
Each problem is its own server under `cmd/`; the implementations live in `internal/`.

| # | Problem | Command |
|---|---|---|
| 0 | Smoke Test | `smoketest` |
| 1 | Prime Time | `primetime` |
| 2 | Means to an End | `means` |
| 3 | Budget Chat | `budgetchat` |
| 4 | Unusual Database Program | `unusualdb` (UDP) |
| 5 | Mob in the Middle | `mobproxy` |
| 6 | Speed Daemon | `speeddaemon` |

```sh
go run ./protohackers/cmd/speeddaemon -port=9999 -log=debug
```
