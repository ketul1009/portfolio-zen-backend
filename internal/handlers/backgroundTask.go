package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"portfolio-zen-backend/internal/database"
	"portfolio-zen-backend/internal/logger"
	"portfolio-zen-backend/internal/responses"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type BackgroundTasksHandler struct {
	db          *database.Client
	logger      *logger.Logger
	redisClient *redis.Client
}

type Job struct {
	ID          string `json:"id"`
	PortfolioID string `json:"portfolio_id"`
	Data        []struct {
		ID        string `json:"id"`
		Symbol    string `json:"symbol"`
		AssetType string `json:"asset_type"`
	} `json:"data"`
}

func NewBackgroundTaskHandler(db *database.Client, logger *logger.Logger, redisClient *redis.Client) *BackgroundTasksHandler {
	return &BackgroundTasksHandler{
		db:          db,
		logger:      logger,
		redisClient: redisClient,
	}
}

func (h *BackgroundTasksHandler) FetchPrices(c *gin.Context) {
	var request struct {
		PortfolioID string `json:"portfolio_id" binding:"required"`
		Symbols     []struct {
			ID        string `json:"id"`
			Symbol    string `json:"symbol"`
			AssetType string `json:"asset_type"`
		} `json:"symbols" binding:"required"`
	}

	body, err := c.GetRawData()
	if err != nil {
		h.logger.Error("Error getting raw data: %v", err)
		responses.SendError(c, http.StatusInternalServerError, "error getting raw data")
		return
	}

	err = json.Unmarshal(body, &request)
	if err != nil {
		h.logger.Error("Error unmarshaling JSON: %v", err)
		responses.SendError(c, http.StatusBadRequest, "error unmarshaling JSON")
		return
	}

	// Add job to Redis queue
	job := Job{
		ID:          uuid.New().String(),
		PortfolioID: request.PortfolioID,
		Data:        request.Symbols,
	}

	// Serialize job to JSON before pushing to Redis
	jobJSON, err := json.Marshal(job)
	if err != nil {
		h.logger.Error("Error marshaling job: %v", err)
		responses.SendError(c, http.StatusInternalServerError, "error creating job")
		return
	}

	h.redisClient.RPush(context.Background(), "jobs", jobJSON)

	responses.SendSuccess(c, "Job added to queue")
}
