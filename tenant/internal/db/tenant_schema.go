package db

import (
	"context"
	"fmt"
	"net/url"

	"github.com/jackc/pgx/v5/pgxpool"
)

// reservedSlugs are PostgreSQL built-in schema names that must not be used as tenant slugs.
var reservedSlugs = map[string]bool{
	"public":             true,
	"pg_catalog":         true,
	"information_schema": true,
	"pg_toast":           true,
	"pg_temp":            true,
}

// IsReservedSlug returns true if the slug collides with a PostgreSQL built-in schema name.
func IsReservedSlug(slug string) bool {
	return reservedSlugs[slug]
}

// ProvisionTenantSchema creates the PostgreSQL schema for a tenant and runs
// all tenant-scoped migrations against it.
// If migrations fail the schema is dropped as a compensating action.
func ProvisionTenantSchema(ctx context.Context, pool *pgxpool.Pool, slug, baseDSN string) error {
	schemaName := "tenant_" + slug

	// Create schema (DDL auto-commits — cannot be inside a transaction)
	if _, err := pool.Exec(ctx, fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %q`, schemaName)); err != nil {
		return fmt.Errorf("create schema %q: %w", schemaName, err)
	}

	// Build a DSN that targets this schema via search_path
	tenantDSN, err := addSearchPath(baseDSN, schemaName)
	if err != nil {
		_ = dropSchema(ctx, pool, schemaName)
		return fmt.Errorf("build tenant dsn: %w", err)
	}

	// Run tenant migrations
	if err := RunTenantMigrations(tenantDSN); err != nil {
		_ = dropSchema(ctx, pool, schemaName)
		return fmt.Errorf("run tenant migrations for %q: %w", schemaName, err)
	}

	return nil
}

// AcquireTenantConn acquires a connection from the pool and sets the search_path
// to the tenant schema so all subsequent queries in that connection are schema-aware.
// Caller MUST call conn.Release() when done.
func AcquireTenantConn(ctx context.Context, pool *pgxpool.Pool, slug string) (*pgxpool.Conn, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire connection: %w", err)
	}

	schemaName := "tenant_" + slug
	if _, err := conn.Exec(ctx, fmt.Sprintf(`SET search_path TO %q, public`, schemaName)); err != nil {
		conn.Release()
		return nil, fmt.Errorf("set search_path to %q: %w", schemaName, err)
	}

	return conn, nil
}

// addSearchPath appends or replaces the search_path query parameter in a DSN.
func addSearchPath(dsn, schemaName string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("search_path", schemaName+",public")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func dropSchema(ctx context.Context, pool *pgxpool.Pool, schemaName string) error {
	_, err := pool.Exec(ctx, fmt.Sprintf(`DROP SCHEMA IF EXISTS %q CASCADE`, schemaName))
	return err
}
