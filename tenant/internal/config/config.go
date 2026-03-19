package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort     string
	AppVersion     string
	GinMode        string
	DatabaseURL    string
	AuthServiceURL string
	AuthStub       bool
	AuthStubClaims StubClaims
}

type StubClaims struct {
	UserID     string
	Role       string
	TenantID   string
	TenantSlug string
	Email      string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading from environment")
	}

	return &Config{
		ServerPort:     getEnv("SERVER_PORT", "8080"),
		AppVersion:     getEnv("APP_VERSION", "1.0.0"),
		GinMode:        getEnv("GIN_MODE", "debug"),
		DatabaseURL:    mustEnv("DATABASE_URL"),
		AuthServiceURL: getEnv("AUTH_SERVICE_URL", "http://localhost:9090"),
		AuthStub:       getBool("AUTH_STUB", false),
		AuthStubClaims: StubClaims{
			UserID:     getEnv("AUTH_STUB_USER_ID", "00000000-0000-0000-0000-000000000001"),
			Role:       getEnv("AUTH_STUB_ROLE", "app_admin"),
			TenantID:   getEnv("AUTH_STUB_TENANT_ID", ""),
			TenantSlug: getEnv("AUTH_STUB_TENANT_SLUG", ""),
			Email:      getEnv("AUTH_STUB_EMAIL", "admin@platform.internal"),
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %q is not set", key)
	}
	return v
}

func getBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
