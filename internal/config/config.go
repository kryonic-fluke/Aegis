package config

import (
	"time"
	"os"
)

type Config struct {
	Port         string
	Env          string        
	LogLevel     string        
	ReadTimeout  time.Duration 
	WriteTimeout time.Duration 
	IdleTimeout  time.Duration 
}

func Load() *Config{
	return &Config{
		Port:         getEnv("PORT", "8080"),
		Env:          getEnv("ENV", "development"),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

func getEnv(key, fallback string) string {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return fallback
	}
	return val
}