package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"codex-mcp/internal/server"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	defaultStatePath, defaultDBPath, defaultProfilePath := defaultPathsFromExecutable()

	cfg := server.Config{
		Logger:           logger,
		StatePath:        envOrDefault("CODEX_TROLLER_STATE_PATH", defaultStatePath),
		DiscussionDBPath: envOrDefault("CODEX_TROLLER_DISCUSSION_DB_PATH", defaultDBPath),
		DefaultProfile:   envOrDefault("CODEX_TROLLER_DEFAULT_PROFILE_PATH", defaultProfilePath),
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

func defaultPathsFromExecutable() (statePath, discussionDBPath, defaultProfilePath string) {
	exePath, err := os.Executable()
	if err != nil {
		statePath = filepath.Join(".codex-mcp", "state", "sessions.json")
		discussionDBPath = filepath.Join(".codex-mcp", "state", "council.db")
		defaultProfilePath = filepath.Join(".codex-mcp", "default_user_profile.json")
		return
	}

	exeDir := filepath.Dir(exePath)
	installRoot := filepath.Dir(exeDir)
	stateDir := filepath.Join(installRoot, "state")

	statePath = filepath.Join(stateDir, "sessions.json")
	discussionDBPath = filepath.Join(stateDir, "council.db")
	defaultProfilePath = filepath.Join(installRoot, "default_user_profile.json")
	return
}
