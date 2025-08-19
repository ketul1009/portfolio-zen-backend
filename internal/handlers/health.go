package handlers

import (
	"net/http"
	"time"

	"portfolio-zen-backend/internal/database"
	"portfolio-zen-backend/internal/logger"
	"portfolio-zen-backend/internal/responses"
	"portfolio-zen-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	db     *database.Client
	broker *services.BrokerService
	logger *logger.Logger
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *database.Client, broker *services.BrokerService, logger *logger.Logger) *HealthHandler {
	return &HealthHandler{
		db:     db,
		broker: broker,
		logger: logger,
	}
}

// HealthCheck handles GET /health requests
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	start := time.Now()

	// Check database health
	dbStatus := "healthy"
	if err := h.db.HealthCheck(); err != nil {
		dbStatus = "unhealthy"
		h.logger.Error("Database health check failed: %v", err)
	}

	// Check broker health
	brokerStatus := "healthy"
	if h.broker != nil {
		if err := h.broker.HealthCheck(); err != nil {
			brokerStatus = "unhealthy"
			h.logger.Error("Broker health check failed: %v", err)
		}
	}

	// Determine overall status
	overallStatus := "healthy"
	if dbStatus == "unhealthy" || brokerStatus == "unhealthy" {
		overallStatus = "unhealthy"
		c.Status(http.StatusServiceUnavailable)
	}

	// Prepare services status
	services := map[string]string{
		"database": dbStatus,
		"broker":   brokerStatus,
	}

	latency := time.Since(start)
	h.logger.LogHealthCheck(overallStatus, latency)

	responses.SendHealthResponse(c, overallStatus, services)
}

// ReadinessCheck handles GET /ready requests
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	// Check if the application is ready to serve requests
	// This is different from health check - it checks if the app is fully initialized

	// Check database connection
	if err := h.db.HealthCheck(); err != nil {
		h.logger.Error("Database not ready: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "not ready",
			"service": "database",
			"error":   err.Error(),
		})
		return
	}

	// Check broker service
	if h.broker != nil {
		if err := h.broker.HealthCheck(); err != nil {
			h.logger.Error("Broker not ready: %v", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "not ready",
				"service": "broker",
				"error":   err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ready",
		"message": "All services are ready",
	})
}

// LivenessCheck handles GET /live requests
func (h *HealthHandler) LivenessCheck(c *gin.Context) {
	// Simple liveness check - just return OK if the process is running
	c.JSON(http.StatusOK, gin.H{
		"status":  "alive",
		"message": "Process is running",
	})
}
