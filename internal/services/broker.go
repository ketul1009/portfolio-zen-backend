package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"portfolio-zen-backend/internal/config"
	"portfolio-zen-backend/internal/database"
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

// makeCoinGeckoRequest makes an HTTP request to CoinGecko API with the API key header
func (bs *BrokerService) makeCoinGeckoRequest(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Add CoinGecko API key header if available
	if bs.config.CoinGeckoAPIKey != "" {
		req.Header.Set("x-cg-demo-api-key", bs.config.CoinGeckoAPIKey)
	}

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	return response, nil
}

// GetCryptoPrice fetches the current price of a cryptocurrency in INR from CoinGecko
func (bs *BrokerService) GetCryptoPrice(symbol string) (float64, error) {
	// Normalize symbol to lowercase for CoinGecko API
	symbolLower := strings.ToLower(symbol)

	// CoinGecko API endpoint using symbols parameter
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?symbols=%s&vs_currencies=inr", symbolLower)

	response, err := bs.makeCoinGeckoRequest(url)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return 0, fmt.Errorf("cryptocurrency not found: %s", symbol)
	}

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return 0, fmt.Errorf("error fetching crypto price: status %d, body: %s", response.StatusCode, string(body))
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, fmt.Errorf("error reading response body: %w", err)
	}

	var responseData map[string]map[string]float64
	if err := json.Unmarshal(body, &responseData); err != nil {
		return 0, fmt.Errorf("error unmarshaling response: %w", err)
	}

	// Check if symbol exists in response
	coinData, exists := responseData[symbolLower]
	if !exists {
		return 0, fmt.Errorf("cryptocurrency not found: %s", symbol)
	}

	// Get INR price
	price, exists := coinData["inr"]
	if !exists {
		return 0, fmt.Errorf("INR price not available for symbol: %s", symbol)
	}

	return price, nil
}

// GetMultipleCryptoPrices fetches prices for multiple cryptocurrencies in INR from CoinGecko
func (bs *BrokerService) GetMultipleCryptoPrices(symbols []string) (map[string]float64, error) {
	if len(symbols) == 0 {
		return make(map[string]float64), nil
	}

	// Normalize symbols to lowercase and maintain mapping to original case
	symbolsLower := make([]string, 0, len(symbols))
	lowerToOriginalMap := make(map[string]string)
	for _, symbol := range symbols {
		symbolLower := strings.ToLower(symbol)
		symbolsLower = append(symbolsLower, symbolLower)
		lowerToOriginalMap[symbolLower] = symbol
	}

	// Join symbols with comma for CoinGecko API
	symbolsStr := strings.Join(symbolsLower, ",")
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?symbols=%s&vs_currencies=inr", symbolsStr)

	response, err := bs.makeCoinGeckoRequest(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("error fetching crypto prices: status %d, body: %s", response.StatusCode, string(body))
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var responseData map[string]map[string]float64
	if err := json.Unmarshal(body, &responseData); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	// Convert response to map[string]float64 using original symbols
	results := make(map[string]float64)
	for symbolLower, coinData := range responseData {
		if originalSymbol, exists := lowerToOriginalMap[symbolLower]; exists {
			if price, priceExists := coinData["inr"]; priceExists {
				results[originalSymbol] = price
			}
		}
	}

	return results, nil
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

// Single function to get LTP of any asset
// TODO: Deprecate other functions
func (bs *BrokerService) GetLTPOfAsset(db *database.Client, assetType string, symbol string, exchange string) (float64, error) {
	if assetType == "stocks" {
		token, err := db.GetToken(symbol)
		if err != nil {
			return 0, fmt.Errorf("error fetching token for symbol: %s", symbol)
		}
		return bs.GetLTP(exchange, symbol, token)
	} else if assetType == "mutual_fund" {
		return bs.GetMutualFundLTP(symbol)
	} else if assetType == "crypto" {
		return bs.GetCryptoPrice(symbol)
	} else {
		return 0, fmt.Errorf("unknown asset type: %s", assetType)
	}
}
