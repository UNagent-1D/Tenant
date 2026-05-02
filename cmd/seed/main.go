package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
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

	var count int
	if err := pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM users WHERE role = 'app_admin'",
	).Scan(&count); err != nil {
		log.Fatalf("query failed: %v", err)
	}
	if count > 0 {
		log.Fatal("app_admin already exists, aborting seed")
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
