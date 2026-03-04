package config

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB() {
	// Puedes establecer la url de conexión mediante variables de entorno, o dejar la default para desarrollo
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		// Fallback local: asume que tienes postgres corriendo en localhost con estos datos
		connStr = "host=localhost port=5432 user=postgres password=admin dbname=postgres sslmode=disable"
	}

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Error al inicializar la base de datos: %q", err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatalf("Error al conectar a la base de datos: %q", err)
	}

	log.Println("Conexión a base de datos exitosa")
}
