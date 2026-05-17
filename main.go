package main

import (
	"log"

	"github.com/UNagent-1D/Tenant/config"
)

func main() {
	// 1. Inicializa la conexión a la base de datos (PostgreSQL)
	config.InitDB()

	// 1b. Idempotent seed for the demo app-admin (skipped if SEED_DEMO_USERS=false).
	seedDemoUsers()

	// 2. Levanta el enrutador de Gin a través del archivo router.go
	router := SetupRouter()

	// 3. Correr servidor
	log.Println("Inciando servidor en el puerto :8080...")
	router.Run(":8080")
}
