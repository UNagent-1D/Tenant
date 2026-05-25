package config

import (
	"os"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL        string
	JWTSecret          string
	JWTExpiry          time.Duration
	InternalAPIKey     string
	HTTPTimeout        time.Duration
	Port               string
	CORSAllowOrigins   []string // space-separated list from CORS_ALLOW_ORIGINS env var
	TrustProxyHeaders  bool     // set true only when behind a trusted reverse proxy (Cloudflare)
}

func Load() *Config {
	return &Config{
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://postgres:admin@localhost:5432/postgres?sslmode=disable"),
		JWTSecret:         getEnv("JWT_SECRET", "super-secreto-cambiar-en-produccion"),
		JWTExpiry:         8 * time.Hour,
		InternalAPIKey:    getEnv("INTERNAL_API_KEY", "dev-internal-key"),
		HTTPTimeout:       10 * time.Second,
		Port:              getEnv("PORT", "8080"),
		CORSAllowOrigins:  parseCORSOrigins(getEnv("CORS_ALLOW_ORIGINS", "http://localhost:3000")),
		TrustProxyHeaders: getEnv("TRUST_PROXY_HEADERS", "false") == "true",
	}
}

func parseCORSOrigins(raw string) []string {
	var out []string
	for _, s := range strings.Fields(raw) {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
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
