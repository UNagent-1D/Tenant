package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// queryInt reads an integer query param with a fallback default.
func queryInt(c *gin.Context, key string, defaultVal int) int {
	s := c.Query(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 {
		return defaultVal
	}
	return v
}
