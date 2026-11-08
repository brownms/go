package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/brownms/go/protohackers/internal/primetime"
	"github.com/brownms/go/protohackers/internal/server"
)

func main() {
	addr := server.ParseFlags()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.ServeTCP(ctx, addr, "primetime", primetime.Handler); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}
