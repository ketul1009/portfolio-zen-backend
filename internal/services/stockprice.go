package services

import (
	"context"
	"fmt"
	"portfolio-zen-backend/internal/tasks"
)

func FetchAndUpdatePrices(ctx context.Context, p tasks.FetchPricePayload) error {
	symbols := p.Symbols
	for _, symbol := range symbols {
		symbolMap := symbol.(map[string]interface{})
		symbolName := symbolMap["symbol"].(string)
		symbolType := symbolMap["asset_type"].(string)

		fmt.Println("Fetching price for", symbolName, "of type", symbolType)
	}
	return nil
}
