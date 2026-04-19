package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/UNagent-1D/Tenant/config"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No se encontró .env, usando variables del sistema")
	}

	config.InitDB()

	router := SetupRouter()

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Iniciando servidor en el puerto :%s...", port)
	router.Run(":" + port)
}

// HealthCheck is defined here to avoid a separate file for a one-liner.
func HealthCheck(c *gin.Context) {
	if err := config.DB.Ping(context.Background()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "db": "error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "db": "ok"})
}
