package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var startTime = time.Now()

type healthResponse struct {
	Status  string  `json:"status"`
	Uptime  float64 `json:"uptime_seconds"`
	Version string  `json:"version,omitempty"`
}

// HealthCheck returns the health status of the service.
func HealthCheck(c *gin.Context) {
	uptime := time.Since(startTime).Seconds()
	c.JSON(http.StatusOK, healthResponse{
		Status:  "healthy",
		Uptime:  uptime,
		Version: "1.0.0",
	})
}

// ReadinessCheck checks if the service is ready to accept traffic.
func ReadinessCheck(c *gin.Context) {
	// In a real application, you would check database connections,
	// external service availability, etc.
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}
