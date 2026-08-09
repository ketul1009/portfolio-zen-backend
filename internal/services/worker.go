package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"portfolio-zen-backend/internal/database"
	"time"

	"github.com/go-redis/redis/v8"
)

type Job struct {
	ID          string `json:"id"`
	PortfolioID string `json:"portfolio_id"`
	Data        []struct {
		Symbol    string `json:"symbol"`
		AssetType string `json:"asset_type"`
	} `json:"data"`
	JobType string `json:"job_type"`
}

type WorkerService struct {
	redisClient *redis.Client
	logger      Logger
}

type Logger interface {
	Info(format string, args ...interface{})
	Error(format string, args ...interface{})
}

func NewWorkerService(redisClient *redis.Client, logger Logger) *WorkerService {
	return &WorkerService{
		redisClient: redisClient,
		logger:      logger,
	}
}

func (w *WorkerService) Start(broker *BrokerService, db *database.Client) {
	ctx := context.Background()
	w.logger.Info("Worker started")

	// Test Redis connection before starting the worker loop
	_, err := w.redisClient.Ping(ctx).Result()
	if err != nil {
		w.logger.Error("Worker failed to connect to Redis: %v", err)
		return
	}
	w.logger.Info("Worker connected to Redis successfully")

	for {
		// BRPOP with 5-second timeout to avoid indefinite blocking
		res, err := w.redisClient.BRPop(ctx, 5*time.Second, "price_update_jobs", "sip_update_jobs").Result()
		if err != nil {
			// Only log if it's not a timeout or EOF error (which are expected when no jobs)
			if err != redis.Nil && !errors.Is(err, io.EOF) {
				w.logger.Error("Error reading from queue: %v", err)
			}
			// Add a small delay before trying again to avoid busy waiting
			time.Sleep(1 * time.Second)
			continue
		}
		if len(res) < 2 {
			continue
		}

		var job Job
		if err := json.Unmarshal([]byte(res[1]), &job); err != nil {
			w.logger.Error("Error decoding job: %v", err)
			continue
		}

		// Process job
		w.logger.Info("Processing job ID=%s with %d symbols", job.ID, len(job.Data))

		if job.JobType == "price_update" {
			BackgroundPriceUpdate(&WorkerDependencies{
				Broker: broker,
				DB:     db,
				Logger: w.logger,
			}, job)
		}

		if job.JobType == "sip_update" {
			BackgroundSIPUpdate(&WorkerDependencies{
				Broker: broker,
				DB:     db,
				Logger: w.logger,
			}, job)
		}
	}
}
