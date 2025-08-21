package handlers

import (
	"net/http"
	"portfolio-zen-backend/internal/database"
	"portfolio-zen-backend/internal/logger"
	"portfolio-zen-backend/internal/responses"

	"github.com/gin-gonic/gin"
)

type MutualFundsHandler struct {
	db     *database.Client
	logger *logger.Logger
}

func NewMutualFundsHandler(db *database.Client, logger *logger.Logger) *MutualFundsHandler {
	return &MutualFundsHandler{
		db:     db,
		logger: logger,
	}
}

func (h *MutualFundsHandler) GetMutualFundNav(c *gin.Context) {
	search_id := c.Param("search_id")
	nav, err := h.db.GetMutualFundNav(search_id)
	if err != nil {
		h.logger.Error("Error getting mutual fund holdings: %v", err)
		responses.SendError(c, http.StatusInternalServerError, "Error getting mutual fund holdings")
		return
	}
	responses.SendSuccess(c, nav)
}
