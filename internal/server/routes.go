package server

import (
	"encoding/json"
	"net/http"
)

func (s *Server) setupRoutes() {
	s.router.Get("/health", s.handleHealth)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{
		"status": "available",
		"env":    s.cfg.Env,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		s.logger.Error("failed to encode health response", "error", err)
	}
}