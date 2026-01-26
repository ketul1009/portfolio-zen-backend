package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"portfolio-zen-backend/internal/database"
	"strings"
	"time"

	"github.com/google/uuid"
)

func GetPortfolios(d *ScheduledJobDependencies, userID string) ([]database.Portfolio, error) {
	var portfolios []database.Portfolio
	portfolios, err := d.DB.GetPortfoliosByUser(userID)
	if err != nil {
		d.Logger.Error("[Cron] [GetPortfolios] Error getting portfolios: %v", err)
		return nil, fmt.Errorf("error getting portfolios: %w", err)
	}
	return portfolios, nil
}

func GetUsersWithAutoUpdate(d *ScheduledJobDependencies) ([]string, error) {
	users, err := d.DB.GetUsersWithAutoUpdate()
	if err != nil {
		d.Logger.Error("[Cron] [GetUsersWithAutoUpdate] Error getting users with auto update: %v", err)
		return nil, fmt.Errorf("error getting users with auto update: %w", err)
	}
	return users, nil
}

func UpdatePortfolio(d *ScheduledJobDependencies) error {
	users, err := GetUsersWithAutoUpdate(d)
	if err != nil {
		d.Logger.Error("[Cron] [UpdatePortfolio] Error getting users with auto update: %v", err)
		return fmt.Errorf("error getting users with auto update: %w", err)
	}

	for _, user := range users {
		portfolios, err := GetPortfolios(d, user)
		if err != nil {
			d.Logger.Error("[Cron] [UpdatePortfolio] Error getting portfolios: %v", err)
			return fmt.Errorf("error getting portfolios: %w", err)
		}
		for _, portfolio := range portfolios {
			FetchPrices(d, user, portfolio.ID)
		}
	}
	return nil
}

func FetchPrices(d *ScheduledJobDependencies, userID string, portfolioID string) error {

	symbols, err := d.DB.GetHoldings(portfolioID)
	if err != nil {
		d.Logger.Error("[Cron] [FetchPrices] Error getting symbols: %v", err)
		return fmt.Errorf("error getting symbols: %w", err)
	}

	var holdings []struct {
		ID        string `json:"id"`
		Symbol    string `json:"symbol"`
		AssetType string `json:"asset_type"`
	}

	for _, symbol := range symbols {
		holdings = append(holdings, struct {
			ID        string `json:"id"`
			Symbol    string `json:"symbol"`
			AssetType string `json:"asset_type"`
		}{
			ID:        symbol.ID,
			Symbol:    symbol.Symbol,
			AssetType: symbol.AssetType,
		})
	}

	// Add job to Redis queue
	job := database.Job{
		ID:          uuid.New().String(),
		UserID:      userID,
		PortfolioID: portfolioID,
		Data:        holdings,
		Status:      "pending",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		JobType:     "price_update",
	}

	// Serialize job to JSON before pushing to Redis
	jobJSON, err := json.Marshal(job)
	if err != nil {
		d.Logger.Error("[Cron] [FetchPrices] Error marshaling job: %v", err)
		return fmt.Errorf("error creating job: %w", err)
	}

	err = d.DB.CreateJob(job)
	if err != nil {
		d.Logger.Error("[Cron] [FetchPrices] Error creating job: %v", err)
		// Check if it's a foreign key constraint error (400) or other error (500)
		if strings.Contains(err.Error(), "does not exist") {
			d.Logger.Error("[Cron] [FetchPrices] Error creating job: %v", err)
			return fmt.Errorf("error creating job: %w", err)
		} else {
			d.Logger.Error("[Cron] [FetchPrices] Error creating job: %v", err)
			return fmt.Errorf("error creating job: %w", err)
		}
	}

	// Push job to Redis queue with error handling
	err = d.RedisClient.RPush(context.Background(), "price_update_jobs", jobJSON).Err()
	if err != nil {
		d.Logger.Error("[Cron] [FetchPrices] Error pushing job to Redis queue: %v", err)
		return fmt.Errorf("failed to add job to queue: %w", err)
	}

	d.Logger.Info("[Cron] [FetchPrices] Successfully added CRON job %s to queue", job.ID)
	return nil
}

func ProcessSips(d *ScheduledJobDependencies) error {
	portfolios, err := d.DB.GetPortfoliosWithSIP()
	if err != nil {
		d.Logger.Error("[Cron] [ProcessSips] Error getting portfolios with SIP: %v", err)
		return fmt.Errorf("error getting portfolios with SIP: %w", err)
	}
	for _, portfolio := range portfolios {
		job := database.Job{
			ID:          uuid.New().String(),
			UserID:      portfolio.UserID,
			PortfolioID: portfolio.ID,
			Data:        nil,
			Status:      "pending",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			JobType:     "sip_update",
		}

		d.DB.CreateJob(job)

		jobJSON, err := json.Marshal(job)
		if err != nil {
			d.Logger.Error("[Cron] [ProcessSips] Error marshaling job: %v", err)
			return fmt.Errorf("error creating job: %w", err)
		}

		err = d.RedisClient.RPush(context.Background(), "sip_update_jobs", jobJSON).Err()
		if err != nil {
			d.Logger.Error("[Cron] [ProcessSips] Error pushing job to Redis queue: %v", err)
			return fmt.Errorf("failed to add job to queue: %w", err)
		}

		d.Logger.Info("[Cron] [ProcessSips] Successfully added CRON job %s to queue", job.ID)
	}
	return nil
}
