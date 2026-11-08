package main

import (
	"log/slog"
	"os"

	"github.com/brownms/go/protohackers/internal/budgetchat"
	"github.com/brownms/go/protohackers/internal/server"
)

func main() {
	addr := server.ParseFlags()
	if err := server.ServeTCP(addr, "budgetchat", budgetchat.Handler); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}
