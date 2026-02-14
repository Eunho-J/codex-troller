package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"codex-mcp/internal/server"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := server.Config{
		Logger:    logger,
		StatePath: ".codex-mcp/state/sessions.json",
	}

	srv := server.NewMCPServer(cfg)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := srv.Run(ctx); err != nil {
		logger.Error("server terminated", "error", err)
		os.Exit(1)
	}
}
