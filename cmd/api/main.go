package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/kryonic-fluke/aegis/internal/config"
	"github.com/kryonic-fluke/aegis/internal/db"
	"github.com/kryonic-fluke/aegis/internal/llm"
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

	ctx := context.Background()

	var pgPool *pgxpool.Pool
	pool, err := db.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Warn("postgres unavailable on boot", "error", err)
	} else {
		pgPool = pool
		defer pgPool.Close()
		logger.Info("connected to postgres with pgvector")
	}

	var redisClient *redis.Client
	rClient, err := db.NewRedisClient(ctx, cfg.RedisURL)
	if err != nil {
		logger.Warn("redis unavailable on boot", "error", err)
	} else {
		redisClient = rClient
		defer redisClient.Close()
		logger.Info("connected to redis")
	}

	provider := llm.NewGeminiProvider(cfg.GeminiAPIKey)

	srv := server.New(cfg, logger, provider, pgPool, redisClient)

	if err := srv.Start(); err != nil {
		logger.Error("server failed to start", "error", err)
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
