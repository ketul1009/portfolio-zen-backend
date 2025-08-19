package main

import (
	"fmt"
	"os"

	SmartApi "github.com/angel-one/smartapigo"
	"github.com/joho/godotenv"
)

// BrokerClient wraps the SmartAPI client
type BrokerClient struct {
	Client *SmartApi.Client
}

// NewBrokerClient creates a new broker client with authentication
func NewBrokerClient() (*BrokerClient, error) {
	// Load .env file
	godotenv.Load()

	clientID := os.Getenv("AONE_USERNAME")
	password := os.Getenv("AONE_PASSWORD")
	apiKey := os.Getenv("AONE_API_KEY")

	ABClient := SmartApi.New(clientID, password, apiKey)

	totpSecret := os.Getenv("AONE_TOKEN")
	totpCode, err := GenerateTOTP(totpSecret)
	if err != nil {
		return nil, fmt.Errorf("error generating TOTP code: %w", err)
	}

	session, err := ABClient.GenerateSession(totpCode)
	if err != nil {
		return nil, fmt.Errorf("error generating session: %w", err)
	}

	// Renew access token
	_, err = ABClient.RenewAccessToken(session.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("error renewing access token: %w", err)
	}

	return &BrokerClient{
		Client: ABClient,
	}, nil
}

// GetLTP fetches the last traded price for a given symbol and token
func (bc *BrokerClient) GetLTP(exchange, tradingSymbol, symbolToken string) (float64, error) {
	ltp, err := bc.Client.GetLTP(SmartApi.LTPParams{
		Exchange:      exchange,
		TradingSymbol: tradingSymbol,
		SymbolToken:   symbolToken,
	})
	if err != nil {
		return 0, fmt.Errorf("error fetching LTP: %w", err)
	}

	return ltp.Ltp, nil
}

// GetLTPByToken fetches the last traded price for a given token (without symbol)
func (bc *BrokerClient) GetLTPByToken(exchange, symbolToken string) (float64, error) {
	ltp, err := bc.Client.GetLTP(SmartApi.LTPParams{
		Exchange:    exchange,
		SymbolToken: symbolToken,
	})
	if err != nil {
		return 0, fmt.Errorf("error fetching LTP: %w", err)
	}

	return ltp.Ltp, nil
}
