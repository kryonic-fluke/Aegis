package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/kryonic-fluke/aegis/internal/cache"
	"github.com/kryonic-fluke/aegis/internal/config"
	"github.com/kryonic-fluke/aegis/internal/db"
	"github.com/kryonic-fluke/aegis/internal/llm"
	"github.com/kryonic-fluke/aegis/internal/ratelimit"
	"github.com/kryonic-fluke/aegis/internal/router"
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

	var limiter *ratelimit.TokenBucketLimiter
	if redisClient != nil {
		limiter = ratelimit.NewTokenBucketLimiter(redisClient, cfg.RateLimitCapacity, cfg.RateLimitRefillRate)
		logger.Info("initialized token bucket rate limiter", "capacity", cfg.RateLimitCapacity, "refill_rate", cfg.RateLimitRefillRate)
	}

	var exactCache *cache.ExactCache
	if redisClient != nil {
		exactCache = cache.NewExactCache(redisClient, cfg.CacheTTL)
		logger.Info("initialized exact cache layer", "ttl", cfg.CacheTTL)
	}

	var embedder llm.Embedder
	if cfg.GeminiAPIKey != "" {
		embedder = llm.NewGeminiEmbedder(cfg.GeminiAPIKey, cfg.EmbeddingModel)
		logger.Info("initialized gemini embedder", "model", cfg.EmbeddingModel)
	}

	var semanticCache *cache.SemanticCache
	if pgPool != nil {
		semanticCache = cache.NewSemanticCache(pgPool)
		if err := semanticCache.InitSchema(ctx); err != nil {
			logger.Warn("failed to initialize semantic cache schema", "error", err)
		} else {
			logger.Info("initialized pgvector semantic cache with hnsw index")
		}
	}

	var modelRouter *router.Router
	if embedder != nil {
		modelRouter = router.NewRouter(embedder, cfg)
		if err := modelRouter.InitPrototypes(ctx); err != nil {
			logger.Warn("failed to initialize router prototypes", "error", err)
		} else {
			logger.Info("initialized hybrid model router with in-memory prototypes")
		}
	}

	provider := llm.NewGeminiProvider(cfg.GeminiAPIKey)

	srv := server.New(cfg, logger, provider, pgPool, redisClient, limiter, exactCache, embedder, semanticCache, modelRouter)

	if err := srv.Start(); err != nil {
		logger.Error("server failed to start", "error", err)
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
