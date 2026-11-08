package main

import (
	"log/slog"
	"os"

	"github.com/brownms/go/protohackers/internal/means"
	"github.com/brownms/go/protohackers/internal/server"
)

func main() {
	addr := server.ParseFlags()
	if err := server.ServeTCP(addr, "means", means.Handler); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}
