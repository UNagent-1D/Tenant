package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthHandler struct {
	db      *pgxpool.Pool
	version string
}

func NewHealthHandler(db *pgxpool.Pool, version string) *HealthHandler {
	return &HealthHandler{db: db, version: version}
}

func (h *HealthHandler) Check(c *gin.Context) {
	dbStatus := "ok"
	overallStatus := "ok"

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		dbStatus = "error: " + err.Error()
		overallStatus = "degraded"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    overallStatus,
		"db":        dbStatus,
		"version":   h.version,
		"timestamp": time.Now().UTC(),
	})
}
