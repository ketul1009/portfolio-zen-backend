package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"portfolio-zen-backend/internal/config"
	"portfolio-zen-backend/internal/database"
	"portfolio-zen-backend/internal/handlers"
	"portfolio-zen-backend/internal/jobs"
	"portfolio-zen-backend/internal/logger"
	"portfolio-zen-backend/internal/middleware"
	"portfolio-zen-backend/internal/router"
	"portfolio-zen-backend/internal/scheduler"
	"portfolio-zen-backend/internal/services"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
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
	sqsClient   *sqs.Client
	sched       *scheduler.Scheduler
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
	if err := a.initRedis(); err != nil {
		return fmt.Errorf("failed to initialize Redis: %w", err)
	}

	// Initialize SQS client
	if err := a.initSQS(); err != nil {
		return fmt.Errorf("failed to initialize SQS: %w", err)
	}

	// Initialize worker service
	workerService := services.NewWorkerService(a.redisClient, a.logger)
	go workerService.Start(a.broker, a.db)

	// Initialize and start scheduler
	a.sched = scheduler.NewScheduler()

	// Register jobs
	jobsDeps := jobs.ScheduledJobDependencies{
		Logger:      a.logger,
		DB:          a.db,
		Broker:      a.broker,
		RedisClient: a.redisClient,
	}
	jobs.RegisterJobs(a.sched, jobsDeps)

	a.sched.Start()
	defer a.sched.Stop()

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

// initRedis initializes the Redis client and validates connection
func (a *App) initRedis() error {
	redisAddr := os.Getenv("REDIS_ADDR")
	redisPassword := os.Getenv("REDIS_PASSWORD")

	if redisAddr == "" {
		return fmt.Errorf("REDIS_ADDR environment variable is required")
	}

	// Build Redis URL - support both local and cloud Redis
	var redisURL string
	if strings.HasPrefix(redisAddr, "rediss://") || strings.HasPrefix(redisAddr, "redis://") {
		// Already a complete URL
		redisURL = redisAddr
	} else {
		// Determine protocol based on environment
		// For local development, use redis:// (non-secure)
		// For production (Upstash), use rediss:// (secure)
		protocol := "redis://"
		if os.Getenv("ENVIRONMENT") == "production" || os.Getenv("FLY_APP_NAME") != "" {
			protocol = "rediss://"
		}

		if redisPassword != "" {
			redisURL = fmt.Sprintf("%sdefault:%s@%s", protocol, redisPassword, redisAddr)
		} else {
			redisURL = fmt.Sprintf("%s%s", protocol, redisAddr)
		}
	}

	// Parse Redis URL using the official method
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	a.redisClient = redis.NewClient(opt)

	// Test Redis connection
	ctx := context.Background()
	_, err = a.redisClient.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis at %s: %w", redisAddr, err)
	}

	a.logger.Info("Successfully connected to Redis at %s", redisAddr)
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
	cryptoHandler := handlers.NewCryptoHandler(a.broker, a.logger)

	// Get active SQS config based on environment
	activeSQSConfig := a.config.GetActiveSQSConfig()
	backgroundTaskHandler := handlers.NewBackgroundTaskHandler(a.db, a.logger, a.redisClient, a.sqsClient, activeSQSConfig.QueueURL, a.broker)

	// Setup routes
	router.SetupRoutes(r, ltpHandler, mutualFundsHandler, healthHandler, backgroundTaskHandler, cryptoHandler)

	return r
}

// initSQS initializes the AWS SQS client
func (a *App) initSQS() error {
	ctx := context.Background()

	// Get active SQS config based on environment
	activeSQSConfig := a.config.GetActiveSQSConfig()

	// Load AWS configuration
	var cfg aws.Config
	var err error

	if activeSQSConfig.AccessKeyID != "" && activeSQSConfig.SecretAccessKey != "" {
		// Use provided credentials
		cfg, err = awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(activeSQSConfig.Region),
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				activeSQSConfig.AccessKeyID,
				activeSQSConfig.SecretAccessKey,
				"",
			)),
		)
	} else {
		// Use default credential chain (IAM roles, etc.)
		cfg, err = awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(activeSQSConfig.Region),
		)
	}

	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	a.sqsClient = sqs.NewFromConfig(cfg)

	// Validate SQS queue URL is set
	if activeSQSConfig.QueueURL == "" {
		return fmt.Errorf("SQS_QUEUE_URL environment variable is required for %s environment", a.config.Environment)
	}

	a.logger.Info("Successfully initialized SQS client for %s environment (region: %s)", a.config.Environment, activeSQSConfig.Region)
	return nil
}
