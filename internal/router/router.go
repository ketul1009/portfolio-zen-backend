package router

import (
	"portfolio-zen-backend/internal/handlers"
	"portfolio-zen-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all the routes for the application
func SetupRoutes(r *gin.Engine, ltpHandler *handlers.LTPHandler, mutualFundsHandler *handlers.MutualFundsHandler, healthHandler *handlers.HealthHandler, backgroundTasksHandler *handlers.BackgroundTasksHandler, cryptoHandler *handlers.CryptoHandler) {
	// API v1 group
	v1 := r.Group("/api/v1")
	{
		// LTP endpoints
		ltp := v1.Group("/ltp")
		{
			ltp.GET("/:symbol", ltpHandler.GetLTP)
			ltp.POST("/batch", ltpHandler.GetMultipleLTP)
		}

		// Mutual Fund endpoints
		mutualFunds := v1.Group("/mutual-funds")
		{
			mutualFunds.GET("/:search_id", mutualFundsHandler.GetMutualFundHoldings)
			mutualFunds.GET("/nav/:symbol", ltpHandler.GetMutualFundLTP)
		}

		// Crypto endpoints
		crypto := v1.Group("/crypto")
		{
			crypto.GET("/price/:symbol", cryptoHandler.GetCryptoPrice)
		}

		// Symbol endpoints
		symbols := v1.Group("/symbols")
		{
			symbols.GET("", ltpHandler.GetAllSymbols)
			symbols.GET("/search", ltpHandler.SearchSymbols)
		}

		// Background tasks endpoints
		backgroundTasks := v1.Group("/background-tasks")
		{
			backgroundTasks.POST("/fetch-prices", backgroundTasksHandler.FetchPrices)
			backgroundTasks.POST("/trigger", backgroundTasksHandler.TriggerScheduledJob)
			backgroundTasks.GET("/job/:job_id", backgroundTasksHandler.GetJobStatus)
			backgroundTasks.POST("/upload-portfolio", backgroundTasksHandler.UploadPortfolio)
		}

		// Protected endpoints (example)
		protected := v1.Group("/protected")
		protected.Use(middleware.Authentication())
		{
			// Add protected endpoints here
			protected.GET("/profile", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Protected endpoint"})
			})
		}
	}

	// Health check endpoints
	health := r.Group("")
	{
		health.GET("/health", healthHandler.HealthCheck)
		health.GET("/ready", healthHandler.ReadinessCheck)
		health.GET("/live", healthHandler.LivenessCheck)
	}

	// Root endpoint
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Portfolio Zen Backend API",
			"version": "1.0.0",
			"status":  "running",
			"docs":    "/api/v1/docs", // TODO: Add API documentation
		})
	})

	// API documentation endpoint (placeholder)
	r.GET("/api/v1/docs", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "API Documentation",
			"endpoints": gin.H{
				"GET /api/v1/ltp/:symbol":    "Get LTP for a specific symbol",
				"POST /api/v1/ltp/batch":     "Get LTP for multiple symbols",
				"GET /api/v1/symbols":        "Get all available symbols",
				"GET /api/v1/symbols/search": "Search symbols by query",
				"GET /health":                "Health check",
				"GET /ready":                 "Readiness check",
				"GET /live":                  "Liveness check",
			},
		})
	})

	// 404 handler for undefined routes
	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{
			"success": false,
			"error":   "Route not found",
			"code":    404,
			"path":    c.Request.URL.Path,
		})
	})
}
