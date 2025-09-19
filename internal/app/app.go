package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"portfolio-zen-backend/internal/config"
	"portfolio-zen-backend/internal/database"
	"portfolio-zen-backend/internal/handlers"
	"portfolio-zen-backend/internal/logger"
	"portfolio-zen-backend/internal/middleware"
	"portfolio-zen-backend/internal/router"
	"portfolio-zen-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// App represents the main application
type App struct {
	config      *config.Config
	logger      *logger.Logger
	db          *database.Client
	broker      *services.BrokerService
	server      *http.Server
	redisClient *redis.Client
}

// New creates a new application instance
func New(cfg *config.Config) *App {
	return &App{
		config: cfg,
	}
}

// Run starts the application
func (a *App) Run() error {
	// Initialize logger
	if err := a.initLogger(); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Initialize database
	if err := a.initDatabase(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer a.db.Close()

	// Initialize broker service
	if err := a.initBrokerService(); err != nil {
		return fmt.Errorf("failed to initialize broker service: %w", err)
	}

	// Initialize Redis client
	a.redisClient = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"), // e.g. "localhost:6379"
	})

	// Initialize worker service
	workerService := services.NewWorkerService(a.redisClient, a.logger)
	go workerService.Start(a.broker, a.db)

	// Initialize router
	router := a.initRouter()

	// Create HTTP server
	a.server = &http.Server{
		Addr:         ":" + a.config.Server.Port,
		Handler:      router,
		ReadTimeout:  a.config.Server.ReadTimeout,
		WriteTimeout: a.config.Server.WriteTimeout,
		IdleTimeout:  a.config.Server.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		a.logger.Info("Starting server on port %s", a.config.Server.Port)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Error("Server failed to start: %v", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	a.logger.Info("Shutting down server...")

	// Give server time to shutdown gracefully
	ctx, cancel := context.WithTimeout(context.Background(), a.config.Server.ShutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		a.logger.Error("Server forced to shutdown: %v", err)
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	a.logger.Info("Server exited gracefully")
	return nil
}

// initLogger initializes the logger
func (a *App) initLogger() error {
	var err error
	a.logger, err = logger.New(a.config.Logging)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}
	return nil
}

// initDatabase initializes the database connection
func (a *App) initDatabase() error {
	var err error
	a.db, err = database.NewClient(a.config.Database)
	if err != nil {
		return fmt.Errorf("failed to create database client: %w", err)
	}
	return nil
}

// initBrokerService initializes the broker service
func (a *App) initBrokerService() error {
	var err error
	a.broker, err = services.NewBrokerService(a.config.Broker)
	if err != nil {
		return fmt.Errorf("failed to create broker service: %w", err)
	}
	return nil
}

// initRouter initializes the router with all middleware and routes
func (a *App) initRouter() *gin.Engine {
	// Set Gin mode
	if a.config.Logging.Level != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	r := gin.New()

	// Add middleware
	r.Use(middleware.Recovery(a.logger))
	r.Use(middleware.CORS())
	r.Use(middleware.RequestLogging(a.logger))
	r.Use(middleware.RateLimiting(a.config.RateLimit, a.logger))

	// Initialize handlers
	ltpHandler := handlers.NewLTPHandler(a.broker, a.db, a.logger)
	mutualFundsHandler := handlers.NewMutualFundsHandler(a.db, a.logger)
	healthHandler := handlers.NewHealthHandler(a.db, a.broker, a.logger)
	backgroundTaskHandler := handlers.NewBackgroundTaskHandler(a.db, a.logger, a.redisClient)

	// Setup routes
	router.SetupRoutes(r, ltpHandler, mutualFundsHandler, healthHandler, backgroundTaskHandler)

	return r
}
