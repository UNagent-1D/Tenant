package tenant

import (
	"context"
	"fmt"
	"net/url"

	"github.com/UNagent-1D/Tenant/config"
	"github.com/UNagent-1D/Tenant/pkg/db"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

var slugBlocklist = map[string]bool{
	"public":             true,
	"pg_catalog":         true,
	"information_schema": true,
}

// provisionTenantSchema creates the per-tenant PostgreSQL schema and runs
// migrations/tenant/ against it. On failure it drops the schema and returns
// the error so the caller can roll back the tenant row.
func provisionTenantSchema(ctx context.Context, slug string, cfg *config.Config) error {
	if slugBlocklist[slug] {
		return fmt.Errorf("slug %q is reserved", slug)
	}

	schemaName := "tenant_" + slug

	if _, err := db.Pool.Exec(ctx, fmt.Sprintf(`CREATE SCHEMA "%s"`, schemaName)); err != nil {
		return fmt.Errorf("already exists")
	}

	migrationsURL := "file://migrations/tenant"

	u, err := url.Parse(cfg.DatabaseURL)
	if err != nil {
		db.Pool.Exec(ctx, fmt.Sprintf(`DROP SCHEMA "%s" CASCADE`, schemaName))
		return fmt.Errorf("invalid DATABASE_URL: %w", err)
	}
	q := u.Query()
	q.Set("search_path", schemaName)
	u.RawQuery = q.Encode()
	dbURL := u.String()

	m, err := migrate.New(migrationsURL, dbURL)
	if err != nil {
		db.Pool.Exec(ctx, fmt.Sprintf(`DROP SCHEMA "%s" CASCADE`, schemaName))
		return fmt.Errorf("migration init failed: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		db.Pool.Exec(ctx, fmt.Sprintf(`DROP SCHEMA "%s" CASCADE`, schemaName))
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}
