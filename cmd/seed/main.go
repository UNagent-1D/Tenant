package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// demoTenantID is the canonical UUID for the single-tenant demo. The FrontEnd
// hardcodes this same value (NEXT_PUBLIC_DEMO_HOSPITAL_TENANT_ID default in
// features/users/UsersManager.tsx) and chat-orch uses it as
// TELEGRAM_DEFAULT_TENANT_ID, so tenant-scoped user creation and Telegram
// enrollment resolve to a row that actually exists on a fresh volume.
const (
	demoTenantID   = "ce5ac1c5-9b16-486a-b091-5468d232a4b8"
	demoTenantSlug = "demo-hospital"
	demoTenantName = "Demo Hospital"
)

func main() {
	email := mustEnv("SEED_EMAIL")
	password := mustEnv("SEED_PASSWORD")
	dbURL := mustEnv("DATABASE_URL")

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("cannot connect to database: %v", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(context.Background(),
		`INSERT INTO tenants (id, slug, name) VALUES ($1, $2, $3)
		 ON CONFLICT (id) DO NOTHING`,
		demoTenantID, demoTenantSlug, demoTenantName,
	); err != nil {
		log.Fatalf("tenant seed failed: %v", err)
	}
	log.Printf("demo tenant ensured: %s (%s)", demoTenantSlug, demoTenantID)

	var count int
	if err := pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM users WHERE role = 'app_admin'",
	).Scan(&count); err != nil {
		log.Fatalf("query failed: %v", err)
	}
	if count > 0 {
		log.Printf("app_admin already exists, skipping admin seed")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		log.Fatalf("bcrypt failed: %v", err)
	}

	if _, err := pool.Exec(context.Background(),
		`INSERT INTO users (email, password_hash, role) VALUES ($1, $2, 'app_admin')`,
		email, string(hash),
	); err != nil {
		log.Fatalf("insert failed: %v", err)
	}

	log.Printf("app_admin created: %s", email)
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required env var: %s", key)
	}
	return v
}
