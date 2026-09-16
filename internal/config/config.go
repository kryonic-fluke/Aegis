package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                   string
	Env                    string
	LogLevel               string
	ReadTimeout            time.Duration
	WriteTimeout           time.Duration
	IdleTimeout            time.Duration
	GeminiAPIKey           string
	DatabaseURL            string
	RedisURL               string
	RateLimitCapacity      int
	RateLimitRefillRate    float64
	CacheTTL               time.Duration
	SemanticCacheThreshold float64
}

func Load() *Config {
	return &Config{
		Port:                   getEnv("PORT", "8080"),
		Env:                    getEnv("ENV", "development"),
		LogLevel:               getEnv("LOG_LEVEL", "info"),
		ReadTimeout:            5 * time.Second,
		WriteTimeout:           10 * time.Second,
		IdleTimeout:            120 * time.Second,
		GeminiAPIKey:           getEnv("GEMINI_API_KEY", ""),
		DatabaseURL:            getEnv("DATABASE_URL", "postgres://aegis_user:aegis_password@localhost:5432/aegis_db?sslmode=disable"),
		RedisURL:               getEnv("REDIS_URL", "localhost:6379"),
		RateLimitCapacity:      getEnvInt("RATE_LIMIT_CAPACITY", 10),
		RateLimitRefillRate:    getEnvFloat("RATE_LIMIT_REFILL_RATE", 2.0),
		CacheTTL:               getEnvDuration("CACHE_TTL", 24*time.Hour),
		SemanticCacheThreshold: getEnvFloat("SEMANTIC_CACHE_THRESHOLD", 0.95),
	}
}

func getEnv(key, fallback string) string {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return fallback
	}
	return val
}

func getEnvInt(key string, fallback int) int {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return fallback
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return i
}

func getEnvFloat(key string, fallback float64) float64 {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return fallback
	}
	return f
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return fallback
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return fallback
	}
	return d
}
