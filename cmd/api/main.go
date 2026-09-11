package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/kryonic-fluke/aegis/internal/config"
	"github.com/kryonic-fluke/aegis/internal/server"
)

func main() {
	cfg := config.Load()

	var handler slog.Handler
	if cfg.Env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	}
	logger := slog.New(handler)

	srv := server.New(cfg, logger)

	
	if err := srv.Start(); err != nil {
		logger.Error("server failed to start", "error", err)
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}