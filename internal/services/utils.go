package services

import (
	"portfolio-zen-backend/internal/database"
	"time"
)

type WorkerDependencies struct {
	Broker *BrokerService
	DB     *database.Client
	Logger Logger
}

func GetServices(d *WorkerDependencies) (*BrokerService, *database.Client, Logger) {
	return d.Broker, d.DB, d.Logger
}

func BackgroundPriceUpdate(d *WorkerDependencies, job Job) {
	broker, db, logger := GetServices(d)
	portfolioID := job.PortfolioID
	portfolio, err := db.GetHoldings(portfolioID)

	if err != nil {
		logger.Error("Error fetching Holdings for Portfolio: %v", err)
		return
	}

	symbols := make([]string, len(portfolio))
	for i, asset := range portfolio {
		symbols[i] = asset.Symbol
	}

	priceData := make(map[string]float64)

	symbolToToken, err := db.GetTokens(symbols)
	if err != nil {
		logger.Error("Error fetching Tokens for Portfolio: %v", err)
		return
	}

	for _, asset := range portfolio {
		token, exists := symbolToToken[asset.Symbol]
		if !exists {
			continue
		}
		if asset.AssetType == "stock" {
			ltp, err := broker.GetLTP("NSE", asset.Symbol, token)
			if err != nil {
				logger.Error("Error fetching LTP for Symbol: %v", err)
				continue
			}
			priceData[asset.Symbol] = ltp
		}
	}

	for _, asset := range portfolio {
		if asset.AssetType == "mutual_fund" {
			ltp, err := broker.GetMutualFundLTP(asset.Symbol)
			if err != nil {
				logger.Error("Error fetching LTP for Symbol: %v", err)
				continue
			}
			priceData[asset.Symbol] = ltp
		}
	}

	// Fetch crypto prices
	cryptoSymbols := make([]string, 0)
	for _, asset := range portfolio {
		if asset.AssetType == "crypto" {
			// CoinGecko API requires lowercase symbols, but we pass original symbol
			// and let the broker service handle the conversion
			cryptoSymbols = append(cryptoSymbols, asset.Symbol)
		}
	}

	if len(cryptoSymbols) > 0 {
		cryptoPrices, err := broker.GetMultipleCryptoPrices(cryptoSymbols)
		if err != nil {
			logger.Error("Error fetching crypto prices: %v", err)
		} else {
			// cryptoPrices map uses original symbols as keys (from broker service)
			for symbol, price := range cryptoPrices {
				priceData[symbol] = price
			}
		}
	}

	logger.Info("Price data: %v", priceData)

	// Bulk update all prices in a single transaction
	if len(priceData) > 0 {
		err := db.BulkUpdateCurrentPrices(priceData)
		if err != nil {
			logger.Error("Error bulk updating prices: %v", err)
		} else {
			logger.Info("Successfully updated prices for %d symbols", len(priceData))
		}
	}

	err = db.UpdateJobStatus(job.ID, "completed")
	if err != nil {
		logger.Error("Error updating job status: %v", err)
	}

	logger.Info("Completed job ID=%s", job.ID)
}

func BackgroundSIPUpdate(d *WorkerDependencies, job Job) {
	broker, db, logger := GetServices(d)
	portfolioID := job.PortfolioID
	portfolio, err := db.GetHoldings(portfolioID)

	if err != nil {
		logger.Error("Error fetching Holdings for Portfolio: %v", err)
		return
	}

	sipData := make(map[string]interface{})

	for _, asset := range portfolio {
		if asset.SipEnabled && asset.NextExecutionDate.Before(time.Now()) {
			ltp, err := broker.GetLTPOfAsset(d.DB, asset.AssetType, asset.Symbol, "NSE")
			if err != nil {
				logger.Error("Error fetching LTP for Symbol: %v", err)
				continue
			}
			quantity := asset.SipAmount / ltp
			newAveragePrice := (asset.PurchasePrice*asset.Quantity + asset.SipAmount) / (asset.Quantity + quantity)
			newQuantity := asset.Quantity + quantity
			sipFrequency := asset.SipFrequency
			newNextExecutionDate := getNextExecutionDate(sipFrequency)

			sipData[asset.ID] = map[string]interface{}{
				"average_price":       newAveragePrice,
				"quantity":            newQuantity,
				"next_execution_date": newNextExecutionDate,
			}
		}
	}

	if len(sipData) > 0 {
		err := db.BulkUpdateAssets(sipData)
		if err != nil {
			logger.Error("Error bulk updating SIP: %v", err)
		} else {
			logger.Info("Successfully updated SIP for %d assets", len(sipData))
		}
	}

	err = db.UpdateJobStatus(job.ID, "completed")
	if err != nil {
		logger.Error("Error updating job status: %v", err)
	}

	logger.Info("Completed job ID=%s", job.ID)
}

func getNextExecutionDate(sipFrequency string) time.Time {
	switch sipFrequency {
	case "daily":
		return time.Now().AddDate(0, 0, 1)
	case "weekly":
		return time.Now().AddDate(0, 0, 7)
	case "monthly":
		return time.Now().AddDate(0, 1, 0)
	case "yearly":
		return time.Now().AddDate(1, 0, 0)
	default:
		return time.Now().AddDate(0, 1, 0)
	}
}
