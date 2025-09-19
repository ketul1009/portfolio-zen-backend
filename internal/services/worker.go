package services

import (
	"context"
	"encoding/json"
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

	for {
		// BRPOP blocks until a job arrives
		res, err := w.redisClient.BRPop(ctx, 0*time.Second, "jobs").Result()
		if err != nil {
			w.logger.Error("Error reading from queue: %v", err)
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
		portfolioID := job.PortfolioID
		portfolio, err := db.GetHoldings(portfolioID)

		if err != nil {
			w.logger.Error("Error fetching Holdings for Portfolio: %v", err)
			continue
		}

		symbols := make([]string, len(portfolio))
		for i, asset := range portfolio {
			symbols[i] = asset.Symbol
		}

		priceData := make(map[string]float64)

		symbolToToken, err := db.GetTokens(symbols)
		if err != nil {
			w.logger.Error("Error fetching Tokens for Portfolio: %v", err)
			continue
		}

		for _, asset := range portfolio {
			token, exists := symbolToToken[asset.Symbol]
			if !exists {
				continue
			}
			if asset.Asset == "stock" {
				ltp, err := broker.GetLTP("NSE", asset.Symbol, token)
				if err != nil {
					w.logger.Error("Error fetching LTP for Symbol: %v", err)
					continue
				}
				priceData[asset.Symbol] = ltp
			}
		}

		for _, asset := range portfolio {
			if asset.Asset == "mutual_fund" {
				ltp, err := broker.GetMutualFundLTP(asset.Symbol)
				if err != nil {
					w.logger.Error("Error fetching LTP for Symbol: %v", err)
					continue
				}
				priceData[asset.Symbol] = ltp
			}
		}

		// Bulk update all prices in a single transaction
		if len(priceData) > 0 {
			err := db.BulkUpdateCurrentPrices(priceData)
			if err != nil {
				w.logger.Error("Error bulk updating prices: %v", err)
			} else {
				w.logger.Info("Successfully updated prices for %d symbols", len(priceData))
			}
		}

		w.logger.Info("Completed job ID=%s", job.ID)
	}
}
