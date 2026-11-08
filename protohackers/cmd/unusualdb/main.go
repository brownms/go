package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/brownms/go/protohackers/internal/server"
	"github.com/brownms/go/protohackers/internal/unusualdb"
)

func main() {
	addr := server.ParseFlags()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.ServeUDP(ctx, addr, "unusualdb", unusualdb.NewStore().Handle); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}
