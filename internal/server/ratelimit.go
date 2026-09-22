package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
)

func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.limiter == nil {
			next.ServeHTTP(w, r)
			return
		}

		clientIP := r.RemoteAddr
		if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			clientIP = host
		}
		if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
			clientIP = realIP
		}

		res, err := s.limiter.Allow(r.Context(), clientIP)
		if err != nil {
			s.logger.Error("rate limiter error", "error", err)
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", s.cfg.RateLimitCapacity))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", res.Remaining))

		if !res.Allowed {
			w.Header().Set("Retry-After", "1")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "too many requests, please slow down",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}
