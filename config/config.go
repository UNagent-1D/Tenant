package config

import (
	"os"
	"time"
)

type Config struct {
	DatabaseURL    string
	JWTSecret      string
	JWTExpiry      time.Duration
	InternalAPIKey string
	HTTPTimeout    time.Duration
	Port           string
}

func Load() *Config {
	return &Config{
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:admin@localhost:5432/postgres?sslmode=disable"),
		JWTSecret:      getEnv("JWT_SECRET", "super-secreto-cambiar-en-produccion"),
		JWTExpiry:      8 * time.Hour,
		InternalAPIKey: mustEnv("INTERNAL_API_KEY"),
		HTTPTimeout:    10 * time.Second,
		Port:           getEnv("PORT", "8080"),
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("missing required env var: " + key)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
