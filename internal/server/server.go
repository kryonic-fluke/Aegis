package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/kryonic-fluke/aegis/internal/cache"
	"github.com/kryonic-fluke/aegis/internal/config"
	"github.com/kryonic-fluke/aegis/internal/llm"
	"github.com/kryonic-fluke/aegis/internal/metrics"
	"github.com/kryonic-fluke/aegis/internal/ratelimit"
	"github.com/kryonic-fluke/aegis/internal/router"
)

type Server struct {
	cfg           *config.Config
	logger        *slog.Logger
	router        *chi.Mux
	provider      llm.Provider
	pgPool        *pgxpool.Pool
	redisClient   *redis.Client
	limiter       *ratelimit.TokenBucketLimiter
	exactCache    *cache.ExactCache
	embedder      llm.Embedder
	semanticCache *cache.SemanticCache
	modelRouter   *router.Router
}

func New(cfg *config.Config, logger *slog.Logger, provider llm.Provider, pgPool *pgxpool.Pool, redisClient *redis.Client, limiter *ratelimit.TokenBucketLimiter, exactCache *cache.ExactCache, embedder llm.Embedder, semanticCache *cache.SemanticCache, modelRouter *router.Router) *Server {
	s := &Server{
		cfg:           cfg,
		logger:        logger,
		router:        chi.NewRouter(),
		provider:      provider,
		pgPool:        pgPool,
		redisClient:   redisClient,
		limiter:       limiter,
		exactCache:    exactCache,
		embedder:      embedder,
		semanticCache: semanticCache,
		modelRouter:   modelRouter,
	}

	s.setupMiddlewares()
	s.setupRoutes()

	return s
}

func (s *Server) setupMiddlewares() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(metrics.Middleware)
	s.router.Use(middleware.Recoverer)
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%s", s.cfg.Port)

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  s.cfg.ReadTimeout,
		WriteTimeout: s.cfg.WriteTimeout,
		IdleTimeout:  s.cfg.IdleTimeout,
	}

	shutdownError := make(chan error)
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		sig := <-quit

		s.logger.Info("shutting down server", "signal", sig.String())

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		shutdownError <- httpServer.Shutdown(ctx)
	}()

	s.logger.Info("server starting", "addr", addr, "env", s.cfg.Env)
	err := httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return <-shutdownError
	}
	return err
}
