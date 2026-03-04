package db

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunGlobalMigrations applies all pending migrations from migrations/global.
func RunGlobalMigrations(dsn string) error {
	m, err := migrate.New("file://migrations/global", dsn)
	if err != nil {
		return fmt.Errorf("init global migrate: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run global migrations: %w", err)
	}
	return nil
}

// RunTenantMigrations applies tenant-scoped migrations against the given schema.
// dsn must already contain search_path=tenant_<slug>,public.
func RunTenantMigrations(dsn string) error {
	m, err := migrate.New("file://migrations/tenant", dsn)
	if err != nil {
		return fmt.Errorf("init tenant migrate: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run tenant migrations: %w", err)
	}
	return nil
}
