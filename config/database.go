package config

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func InitDB() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:admin@localhost:5432/tenantdb?sslmode=disable"
	}

	var err error
	DB, err = pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Error al inicializar la base de datos: %v", err)
	}

	if err = DB.Ping(context.Background()); err != nil {
		log.Fatalf("Error al conectar a la base de datos: %v", err)
	}

	log.Println("Conexión a base de datos exitosa")
}
