package handlers

import (
	"encoding/json"
	"io"
	"portfolio-zen-backend/internal/logger"
	"portfolio-zen-backend/internal/tasks"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
)

type TaskHandler struct {
	client *asynq.Client
	logger *logger.Logger
}

func NewTaskHandler(client *asynq.Client, logger *logger.Logger) *TaskHandler {
	return &TaskHandler{client: client, logger: logger}
}

func (h *TaskHandler) FetchPrices(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("Failed to read request body", err)
	}

	var payload tasks.FetchPricePayload
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.Error("Failed to unmarshal request body", err)
	}

	task, err := tasks.NewFetchPriceTask("1", payload.Symbols)
	if err != nil {
		h.logger.Error("Failed to create task", err)
	}

	if _, err := h.client.Enqueue(task); err != nil {
		h.logger.Error("Failed to enqueue task", err)
	}
}
