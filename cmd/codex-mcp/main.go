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
		Logger:           logger,
		StatePath:        envOrDefault("CODEX_TROLLER_STATE_PATH", ".codex-mcp/state/sessions.json"),
		DiscussionDBPath: envOrDefault("CODEX_TROLLER_DISCUSSION_DB_PATH", ""),
		DefaultProfile:   envOrDefault("CODEX_TROLLER_DEFAULT_PROFILE_PATH", ""),
	}

	srv := server.NewMCPServer(cfg)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := srv.Run(ctx); err != nil {
		logger.Error("server terminated", "error", err)
		os.Exit(1)
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
