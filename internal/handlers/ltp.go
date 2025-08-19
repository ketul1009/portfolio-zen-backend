package handlers

import (
	"fmt"
	"net/http"
	"time"

	"portfolio-zen-backend/internal/database"
	"portfolio-zen-backend/internal/logger"
	"portfolio-zen-backend/internal/responses"
	"portfolio-zen-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// LTPHandler handles LTP-related requests
type LTPHandler struct {
	broker *services.BrokerService
	db     *database.Client
	logger *logger.Logger
}

// NewLTPHandler creates a new LTP handler
func NewLTPHandler(broker *services.BrokerService, db *database.Client, logger *logger.Logger) *LTPHandler {
	return &LTPHandler{
		broker: broker,
		db:     db,
		logger: logger,
	}
}

// GetLTP handles GET /api/ltp/:symbol requests
func (h *LTPHandler) GetLTP(c *gin.Context) {
	start := time.Now()
	symbol := c.Param("symbol")

	if symbol == "" {
		h.logger.Error("Symbol parameter is required")
		responses.SendError(c, http.StatusBadRequest, "symbol parameter is required")
		return
	}

	// Get token for symbol
	token, err := h.db.GetTokenForSymbol(symbol)
	if err != nil {
		h.logger.Error("Symbol not found: %s", symbol)
		responses.SendError(c, http.StatusNotFound, fmt.Sprintf("symbol not found: %s", symbol))
		return
	}

	// Fetch LTP
	ltp, err := h.broker.GetLTP("NSE", symbol, token)
	if err != nil {
		h.logger.LogLTPError(symbol, token, err)
		responses.SendError(c, http.StatusInternalServerError, fmt.Sprintf("error fetching LTP: %s", err.Error()))
		return
	}

	// Log successful LTP request
	latency := time.Since(start)
	h.logger.LogLTPRequest(symbol, token, ltp, latency)

	// Return successful response
	responses.SendLTPResponse(c, symbol, token, ltp)
}

// GetMultipleLTP handles POST /api/ltp/batch requests
func (h *LTPHandler) GetMultipleLTP(c *gin.Context) {
	var request struct {
		Symbols []string `json:"symbols" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.Error("Invalid request body: %v", err)
		responses.SendError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(request.Symbols) == 0 {
		responses.SendError(c, http.StatusBadRequest, "symbols array cannot be empty")
		return
	}

	if len(request.Symbols) > 50 {
		responses.SendError(c, http.StatusBadRequest, "maximum 50 symbols allowed per request")
		return
	}

	// Get tokens for all symbols
	symbols := make([]string, 0)
	tokens := make([]string, 0)

	for _, symbol := range request.Symbols {
		token, err := h.db.GetTokenForSymbol(symbol)
		if err != nil {
			h.logger.Error("Symbol not found: %s", symbol)
			continue
		}
		symbols = append(symbols, symbol)
		tokens = append(tokens, token)
	}

	if len(symbols) == 0 {
		responses.SendError(c, http.StatusNotFound, "no valid symbols found")
		return
	}

	// Fetch LTP for all symbols
	ltpResults, err := h.broker.GetMultipleLTP("NSE", symbols, tokens)
	if err != nil {
		h.logger.Error("Error fetching multiple LTP: %v", err)
		responses.SendError(c, http.StatusInternalServerError, "error fetching LTP data")
		return
	}

	// Return successful response
	responses.SendSuccess(c, ltpResults)
}

// SearchSymbols handles GET /api/symbols/search requests
func (h *LTPHandler) SearchSymbols(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		responses.SendError(c, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	if len(query) < 2 {
		responses.SendError(c, http.StatusBadRequest, "query must be at least 2 characters long")
		return
	}

	symbols, err := h.db.SearchSymbols(query)
	if err != nil {
		h.logger.Error("Error searching symbols: %v", err)
		responses.SendError(c, http.StatusInternalServerError, "error searching symbols")
		return
	}

	responses.SendSuccess(c, symbols)
}

// GetAllSymbols handles GET /api/symbols requests
func (h *LTPHandler) GetAllSymbols(c *gin.Context) {
	symbols, err := h.db.GetAllSymbols()
	if err != nil {
		h.logger.Error("Error fetching all symbols: %v", err)
		responses.SendError(c, http.StatusInternalServerError, "error fetching symbols")
		return
	}

	responses.SendSuccess(c, symbols)
}
