package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	godotenv.Load()

	// Set Gin to release mode in production
	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create logger
	logger := NewLogger()

	// Create router
	router := gin.Default()

	// Add CORS middleware
	router.Use(cors.Default())

	// Add request logging middleware
	router.Use(func(c *gin.Context) {
		logger.LogRequest(c)
	})

	// Create rate limiter
	rateLimiter := NewRateLimiter()

	// Use rate limiting middleware for all routes
	router.Use(rateLimiter.RateLimitMiddleware())

	// Create broker client
	brokerClient, err := NewBrokerClient()
	if err != nil {
		logger.LogError("Error creating broker client: %v", err)
		log.Fatal("Error creating broker client:", err)
	}

	// Create database client
	dbClient, err := NewDBClient()
	if err != nil {
		logger.LogError("Error creating database client: %v", err)
		log.Fatal("Error creating database client:", err)
	}
	defer dbClient.Close()

	// Add a simple health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// Add LTP endpoint
	router.GET("/api/ltp/:symbol", func(c *gin.Context) {
		start := time.Now()
		symbol := c.Param("symbol")
		if symbol == "" {
			logger.LogError("Symbol parameter is required")
			SendError(c, http.StatusBadRequest, "symbol parameter is required")
			return
		}

		// Get token for symbol
		token, err := dbClient.GetTokenForSymbol(symbol)
		if err != nil {
			logger.LogError("Symbol not found: %s", symbol)
			SendError(c, http.StatusNotFound, fmt.Sprintf("symbol not found: %s", symbol))
			return
		}

		// Fetch LTP
		ltp, err := brokerClient.GetLTP("NSE", symbol, token)
		if err != nil {
			logger.LogLTPError(symbol, token, err)
			SendError(c, http.StatusInternalServerError, fmt.Sprintf("error fetching LTP: %s", err.Error()))
			return
		}

		// Log successful LTP request
		latency := time.Since(start)
		logger.LogLTPRequest(symbol, token, ltp, latency)

		// Return successful response
		SendLTPResponse(c, symbol, token, ltp)
	})

	// Start server in a goroutine
	server := &http.Server{
		Addr:    ":8000",
		Handler: router,
	}

	// Run server in a goroutine
	go func() {
		fmt.Println("Starting server on :8000")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down server...")

	// Give server 5 seconds to shutdown gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	fmt.Println("Server exited")
}
