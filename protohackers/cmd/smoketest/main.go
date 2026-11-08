package main

import (
	"log/slog"
	"os"

	"github.com/brownms/go/protohackers/internal/server"
	"github.com/brownms/go/protohackers/internal/smoketest"
)

func main() {
	addr := server.ParseFlags()
	if err := server.ServeTCP(addr, "smoketest", smoketest.Handler); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}
