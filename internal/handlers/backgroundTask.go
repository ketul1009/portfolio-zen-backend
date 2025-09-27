package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"portfolio-zen-backend/internal/database"
	"portfolio-zen-backend/internal/logger"
	"portfolio-zen-backend/internal/responses"
	"strings"
	"time"

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
	FileURL string `json:"file_url"`
	JobType string `json:"job_type"`
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
		UserID      string `json:"user_id" binding:"required"`
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
	job := database.Job{
		ID:          uuid.New().String(),
		UserID:      request.UserID,
		PortfolioID: request.PortfolioID,
		Data:        request.Symbols,
		Status:      "pending",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		JobType:     "price_update",
	}

	// Serialize job to JSON before pushing to Redis
	jobJSON, err := json.Marshal(job)
	if err != nil {
		h.logger.Error("Error marshaling job: %v", err)
		responses.SendError(c, http.StatusInternalServerError, "error creating job")
		return
	}

	err = h.db.CreateJob(job)
	if err != nil {
		h.logger.Error("Error creating job: %v", err)
		// Check if it's a foreign key constraint error (400) or other error (500)
		if strings.Contains(err.Error(), "does not exist") {
			responses.SendError(c, http.StatusBadRequest, err.Error())
		} else {
			responses.SendError(c, http.StatusInternalServerError, "error creating job")
		}
		return
	}

	// Push job to Redis queue with error handling
	err = h.redisClient.RPush(context.Background(), "price_update_jobs", jobJSON).Err()
	if err != nil {
		h.logger.Error("Error pushing job to Redis queue: %v", err)
		responses.SendError(c, http.StatusInternalServerError, "failed to add job to queue")
		return
	}

	h.logger.Info("Successfully added job %s to queue", job.ID)
	responses.SendSuccess(c, map[string]string{"message": "Job added to queue", "job_id": job.ID})
}

func (h *BackgroundTasksHandler) GetJobStatus(c *gin.Context) {
	jobID := c.Param("job_id")
	job, err := h.db.GetJob(jobID)
	if err != nil {
		h.logger.Error("Error getting job status: %v", err)
		responses.SendError(c, http.StatusInternalServerError, "failed to get job status")
		return
	}
	responses.SendSuccess(c, map[string]string{"status": job.Status})
}

func (h *BackgroundTasksHandler) UploadPortfolio(c *gin.Context) {
	var request struct {
		UserID      string `form:"user_id" binding:"required"`
		PortfolioID string `form:"portfolio_id" binding:"required"`
		FileURL     string `form:"file_url" binding:"required"`
	}

	err := c.ShouldBind(&request)
	if err != nil {
		h.logger.Error("Error binding form data: %v", err)
		responses.SendError(c, http.StatusBadRequest, "error binding form data")
		return
	}

	// Add job to Redis queue
	job := database.Job{
		ID:          uuid.New().String(),
		UserID:      request.UserID,
		PortfolioID: request.PortfolioID,
		Status:      "pending",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		FileURL:     request.FileURL,
		JobType:     "upload_portfolio",
	}

	// Serialize job to JSON before pushing to Redis
	jobJSON, err := json.Marshal(job)
	if err != nil {
		h.logger.Error("Error marshaling job: %v", err)
		responses.SendError(c, http.StatusInternalServerError, "error creating job")
		return
	}

	err = h.db.CreateJob(job)
	if err != nil {
		h.logger.Error("Error creating job: %v", err)
		// Check if it's a foreign key constraint error (400) or other error (500)
		if strings.Contains(err.Error(), "does not exist") {
			responses.SendError(c, http.StatusBadRequest, err.Error())
		} else {
			responses.SendError(c, http.StatusInternalServerError, "error creating job")
		}
		return
	}

	err = h.redisClient.RPush(context.Background(), "upload_portfolio_jobs", jobJSON).Err()
	if err != nil {
		h.logger.Error("Error pushing job to Redis queue: %v", err)
		responses.SendError(c, http.StatusInternalServerError, "failed to add job to queue")
		return
	}

	h.logger.Info("Successfully added job %s to queue", job.ID)
	responses.SendSuccess(c, map[string]string{"message": "Job added to queue", "job_id": job.ID})
}
