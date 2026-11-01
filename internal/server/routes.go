package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *Server) setupRoutes() {
	s.router.Get("/", s.handleIndex)
	s.router.Get("/health", s.handleHealth)
	s.router.Handle("/metrics", promhttp.Handler())

	s.router.Route("/v1", func(r chi.Router) {
		r.Use(s.rateLimitMiddleware)
		r.Post("/chat/completions", s.handleChatCompletions)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	pgStatus := "up"
	if s.pgPool != nil {
		if err := s.pgPool.Ping(ctx); err != nil {
			pgStatus = "down"
		}
	} else {
		pgStatus = "unconfigured"
	}

	redisStatus := "up"
	if s.redisClient != nil {
		if err := s.redisClient.Ping(ctx).Err(); err != nil {
			redisStatus = "down"
		}
	} else {
		redisStatus = "unconfigured"
	}

	status := "available"
	statusCode := http.StatusOK
	if pgStatus == "down" || redisStatus == "down" {
		status = "degraded"
		statusCode = http.StatusServiceUnavailable
	}

	resp := map[string]string{
		"status":   status,
		"env":      s.cfg.Env,
		"postgres": pgStatus,
		"redis":    redisStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		s.logger.Error("failed to encode health response", "error", err)
	}
}
