package main

import (
	"log"
	"os"
	"strings"

	"github.com/UNagent-1D/Tenant/config"
	"golang.org/x/crypto/bcrypt"
)

// seedDemoUsers idempotently creates the platform's demo app-admin so the
// dashboard works the first time someone hits a fresh database. Runs every
// boot — INSERTs use ON CONFLICT DO NOTHING so it's safe to repeat. Skipped
// entirely when SEED_DEMO_USERS=false (set in environments that prefer a
// stricter empty state).
func seedDemoUsers() {
	if strings.EqualFold(os.Getenv("SEED_DEMO_USERS"), "false") {
		return
	}

	password := os.Getenv("SEED_DEMO_PASSWORD")
	if password == "" {
		password = "demo1234"
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("seed: bcrypt hash failed: %v", err)
		return
	}

	emails := []string{"admin@demo.com", "admin@demo.local"}
	for _, email := range emails {
		var userID string
		err := config.DB.QueryRow(
			`INSERT INTO users (email, password_hash, first_name, last_name)
			 VALUES ($1, $2, 'Demo', 'Admin')
			 ON CONFLICT (email) DO UPDATE
			   SET password_hash = EXCLUDED.password_hash
			 RETURNING id`,
			email, string(hash),
		).Scan(&userID)
		if err != nil {
			log.Printf("seed: upsert %s failed: %v", email, err)
			continue
		}

		if _, err := config.DB.Exec(
			`INSERT INTO user_tenants (user_id, tenant_id, role)
			 VALUES ($1, NULL, 'app_admin')
			 ON CONFLICT (user_id, tenant_id) DO NOTHING`,
			userID,
		); err != nil {
			log.Printf("seed: link %s as app_admin failed: %v", email, err)
		}
	}
	log.Println("seed: demo app-admin users present")
}
