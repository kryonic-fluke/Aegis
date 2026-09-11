package server

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/kryonic-fluke/aegis/internal/config"
)

type Server struct {
	cfg    *config.Config
	logger *slog.Logger
	router *chi.Mux
}

func New(cfg *config.Config, logger *slog.Logger) *Server {
	s := &Server{
		cfg:    cfg,
		logger: logger,
		router: chi.NewRouter(),
	}

	s.setupMiddlewares()
	s.setupRoutes()

	return s
}

func (s *Server) setupMiddlewares() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
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

	s.logger.Info("server starting", "addr", addr, "env", s.cfg.Env)
	return httpServer.ListenAndServe()
}