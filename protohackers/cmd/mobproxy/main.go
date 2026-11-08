package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/brownms/go/protohackers/internal/mobproxy"
	"github.com/brownms/go/protohackers/internal/server"
)

func main() {
	upstream := flag.String("upstream", mobproxy.DefaultUpstream, "Budget Chat server to proxy to")
	addr := server.ParseFlags()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.ServeTCP(ctx, addr, "mobproxy", mobproxy.New(*upstream).Handle); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}
