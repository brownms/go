package main

import (
	"log/slog"
	"os"

	"github.com/brownms/go/protohackers/internal/server"
	"github.com/brownms/go/protohackers/internal/unusualdb"
)

func main() {
	addr := server.ParseFlags()
	if err := server.ServeUDP(addr, "unusualdb", unusualdb.Handler); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}
