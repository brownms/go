package main

import (
	"log/slog"
	"os"

	"github.com/brownms/go/protohackers/internal/mobproxy"
	"github.com/brownms/go/protohackers/internal/server"
)

func main() {
	addr := server.ParseFlags()
	if err := server.ServeTCP(addr, "mobproxy", mobproxy.Handler); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}
