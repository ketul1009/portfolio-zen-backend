package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"portfolio-zen-backend/internal/logger"
	"portfolio-zen-backend/internal/responses"
	"portfolio-zen-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// CryptoHandler handles cryptocurrency-related requests
type CryptoHandler struct {
	broker *services.BrokerService
	logger *logger.Logger
}

// NewCryptoHandler creates a new crypto handler
func NewCryptoHandler(broker *services.BrokerService, logger *logger.Logger) *CryptoHandler {
	return &CryptoHandler{
		broker: broker,
		logger: logger,
	}
}

// GetCryptoPrice handles GET /crypto/price/:symbol requests
func (h *CryptoHandler) GetCryptoPrice(c *gin.Context) {
	symbol := c.Param("symbol")

	if symbol == "" {
		h.logger.Error("Symbol parameter is required")
		responses.SendError(c, http.StatusBadRequest, "symbol parameter is required")
		return
	}

	// Normalize symbol to lowercase for CoinGecko API
	symbolLower := strings.ToLower(symbol)

	// Fetch crypto price from CoinGecko
	price, err := h.broker.GetCryptoPrice(symbolLower)
	if err != nil {
		h.logger.Error("Error fetching crypto price for symbol %s: %v", symbol, err)
		// Check if it's a not found error
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "invalid") {
			c.JSON(http.StatusNotFound, gin.H{
				"error":  "Cryptocurrency not found",
				"code":   "CRYPTO_NOT_FOUND",
				"symbol": symbol,
			})
			return
		}
		responses.SendError(c, http.StatusInternalServerError, fmt.Sprintf("error fetching crypto price: %s", err.Error()))
		return
	}

	// Return successful response with standardized format
	responses.SendSuccess(c, gin.H{
		"symbol": symbol,
		"price":  price,
	})
}
