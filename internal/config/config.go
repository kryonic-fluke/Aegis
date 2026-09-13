package config

import (
	"os"
	"time"
)

type Config struct {
	Port         string
	Env          string
	LogLevel     string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	GeminiAPIKey string
	DatabaseURL  string
	RedisURL     string
}

func Load() *Config {
	return &Config{
		Port:         getEnv("PORT", "8080"),
		Env:          getEnv("ENV", "development"),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		GeminiAPIKey: getEnv("GEMINI_API_KEY", ""),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://aegis_user:aegis_password@localhost:5432/aegis_db?sslmode=disable"),
		RedisURL:     getEnv("REDIS_URL", "localhost:6379"),
	}
}

func getEnv(key, fallback string) string {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return fallback
	}
	return val
}
