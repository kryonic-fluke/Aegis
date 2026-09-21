package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aegis_requests_total",
			Help: "Total number of HTTP requests processed by Aegis",
		},
		[]string{"method", "route", "status", "cache"},
	)

	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "aegis_request_duration_seconds",
			Help:    "Histogram of request latencies in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0},
		},
		[]string{"method", "route", "cache"},
	)

	RateLimitBlocksTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "aegis_rate_limit_blocks_total",
			Help: "Total number of requests blocked by the rate limiter",
		},
	)

	FailoverEventsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "aegis_failover_events_total",
			Help: "Total number of failover events triggered to secondary provider",
		},
	)
)

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rec, r)

		duration := time.Since(start).Seconds()
		routePattern := chi.RouteContext(r.Context()).RoutePattern()
		if routePattern == "" {
			routePattern = r.URL.Path
		}

		cacheStatus := w.Header().Get("X-Cache")
		if cacheStatus == "" {
			cacheStatus = "NONE"
		}

		statusStr := strconv.Itoa(rec.statusCode)
		RequestsTotal.WithLabelValues(r.Method, routePattern, statusStr, cacheStatus).Inc()
		RequestDuration.WithLabelValues(r.Method, routePattern, cacheStatus).Observe(duration)

		if rec.statusCode == http.StatusTooManyRequests {
			RateLimitBlocksTotal.Inc()
		}

		if w.Header().Get("X-Provider-Failover") == "true" {
			FailoverEventsTotal.Inc()
		}
	})
}
