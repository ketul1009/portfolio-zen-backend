package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"portfolio-zen-backend/internal/config"
	"portfolio-zen-backend/internal/logger"
	"portfolio-zen-backend/internal/totp"

	SmartApi "github.com/angel-one/smartapigo"
)

// BrokerService wraps the SmartAPI client with additional functionality
type BrokerService struct {
	client *SmartApi.Client
	config config.BrokerConfig
	logger *logger.Logger
}

// NewBrokerService creates a new broker service
func NewBrokerService(cfg config.BrokerConfig) (*BrokerService, error) {
	client := SmartApi.New(cfg.Username, cfg.Password, cfg.APIKey)

	// Generate TOTP code
	totpCode, err := totp.GenerateTOTP(cfg.TOTPSecret)
	if err != nil {
		return nil, fmt.Errorf("error generating TOTP code: %w", err)
	}

	// Generate session
	session, err := client.GenerateSession(totpCode)
	if err != nil {
		return nil, fmt.Errorf("error generating session: %w", err)
	}

	// Renew access token
	_, err = client.RenewAccessToken(session.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("error renewing access token: %w", err)
	}

	service := &BrokerService{
		client: client,
		config: cfg,
	}

	return service, nil
}

// SetLogger sets the logger for the broker service
func (bs *BrokerService) SetLogger(l *logger.Logger) {
	bs.logger = l
}

// GetLTP fetches the last traded price for a given symbol and token
func (bs *BrokerService) GetLTP(exchange, tradingSymbol, symbolToken string) (float64, error) {
	start := time.Now()

	ltp, err := bs.client.GetLTP(SmartApi.LTPParams{
		Exchange:      exchange,
		TradingSymbol: tradingSymbol,
		SymbolToken:   symbolToken,
	})
	if err != nil {
		if bs.logger != nil {
			bs.logger.LogBrokerError("GetLTP", err)
		}
		return 0, fmt.Errorf("error fetching LTP: %w", err)
	}

	latency := time.Since(start)
	if bs.logger != nil {
		bs.logger.LogLTPRequest(tradingSymbol, symbolToken, ltp.Ltp, latency)
	}

	return ltp.Ltp, nil
}

// GetLTPByToken fetches the last traded price for a given token (without symbol)
func (bs *BrokerService) GetLTPByToken(exchange, symbolToken string) (float64, error) {
	start := time.Now()

	ltp, err := bs.client.GetLTP(SmartApi.LTPParams{
		Exchange:    exchange,
		SymbolToken: symbolToken,
	})
	if err != nil {
		if bs.logger != nil {
			bs.logger.LogBrokerError("GetLTPByToken", err)
		}
		return 0, fmt.Errorf("error fetching LTP: %w", err)
	}

	latency := time.Since(start)
	if bs.logger != nil {
		bs.logger.LogLTPRequest("", symbolToken, ltp.Ltp, latency)
	}

	return ltp.Ltp, nil
}

// GetMultipleLTP fetches LTP for multiple symbols
func (bs *BrokerService) GetMultipleLTP(exchange string, symbols []string, tokens []string) (map[string]float64, error) {
	if len(symbols) != len(tokens) {
		return nil, fmt.Errorf("symbols and tokens arrays must have the same length")
	}

	results := make(map[string]float64)
	errors := make([]string, 0)

	for i, symbol := range symbols {
		ltp, err := bs.GetLTP(exchange, symbol, tokens[i])
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", symbol, err))
			continue
		}
		results[symbol] = ltp
	}

	if len(errors) > 0 && len(errors) == len(symbols) {
		return nil, fmt.Errorf("all LTP requests failed: %v", errors)
	}

	return results, nil
}

func (bs *BrokerService) GetMutualFundLTP(symbol string) (float64, error) {
	url := "https://mf.captnemo.in/nav/" + symbol
	response, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("error fetching LTP: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, fmt.Errorf("error reading response body: %w", err)
	}

	var responseData map[string]interface{}
	if err := json.Unmarshal(body, &responseData); err != nil {
		return 0, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return responseData["nav"].(float64), nil
}

// HealthCheck checks if the broker service is healthy
func (bs *BrokerService) HealthCheck() error {
	// Try to get a simple LTP request to test connectivity
	// This is a basic health check - you might want to implement a more sophisticated one
	_, err := bs.client.GetLTP(SmartApi.LTPParams{
		Exchange:      "NSE",
		TradingSymbol: "RELIANCE",
		SymbolToken:   "2885",
	})

	if err != nil {
		return fmt.Errorf("broker health check failed: %w", err)
	}

	return nil
}
