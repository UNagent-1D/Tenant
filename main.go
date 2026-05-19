package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/UNagent-1D/Tenant/config"
	"github.com/UNagent-1D/Tenant/pkg/db"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	if err := db.Init(cfg.DatabaseURL); err != nil {
		log.Fatalf("cannot connect to database: %v", err)
	}
	defer db.Close()

	router := SetupRouter(cfg)

	log.Printf("starting server on :%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func HealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := db.Pool.Ping(ctx); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":    "degraded",
			"db":        "error: " + err.Error(),
			"version":   "2.2.0",
			"timestamp": time.Now().UTC(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"db":        "ok",
		"version":   "2.2.0",
		"timestamp": time.Now().UTC(),
	})
}
