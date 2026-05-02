package config

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "host=localhost port=5432 user=postgres password=admin dbname=postgres sslmode=disable"
	}

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Error al inicializar la base de datos: %q", err)
	}

	// Supabase Session Pooler closes idle connections after ~5-10 minutes.
	// Recycle our pool entries before that window so we never hand out a
	// connection the server has already torn down.
	DB.SetConnMaxLifetime(4 * time.Minute)
	DB.SetConnMaxIdleTime(2 * time.Minute)
	DB.SetMaxOpenConns(20)
	DB.SetMaxIdleConns(5)

	if err := DB.Ping(); err != nil {
		log.Fatalf("Error al conectar a la base de datos: %q", err)
	}

	go keepAlive(DB)

	log.Println("Conexión a base de datos exitosa")
}

// keepAlive pings the pool periodically so lib/pq's stale connections are
// evicted proactively, instead of surfacing as a 500 on the first request
// after an idle window.
func keepAlive(db *sql.DB) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := db.PingContext(ctx); err != nil {
			log.Printf("db keepalive ping failed: %v", err)
		}
		cancel()
	}
}
