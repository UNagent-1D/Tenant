package db

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func Init(databaseURL string) error {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return err
	}

	// Tune below Supabase Session Pooler's idle timeout (~5-10 min) so we
	// never hand out a connection the server has already torn down.
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 4 * time.Minute
	cfg.MaxConnIdleTime = 2 * time.Minute
	// pgxpool's built-in health check replaces the manual keepalive goroutine.
	cfg.HealthCheckPeriod = 60 * time.Second

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return err
	}
	if err := pool.Ping(context.Background()); err != nil {
		return err
	}
	Pool = pool
	log.Println("database pool ready")
	return nil
}

func Close() {
	if Pool != nil {
		Pool.Close()
	}
}
