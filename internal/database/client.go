package database

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"portfolio-zen-backend/internal/config"
	"portfolio-zen-backend/internal/logger"

	_ "github.com/lib/pq"
)

// Client wraps the database connection with additional functionality
type Client struct {
	DB     *sql.DB
	config config.DatabaseConfig
	logger *logger.Logger
}

// SymbolToken represents a symbol and its corresponding token
type SymbolToken struct {
	Symbol string `json:"symbol"`
	Token  string `json:"token"`
}

// AssetType
type Asset struct {
	ID     string `json:"id"`
	Asset  string `json:"asset"`
	Symbol string `json:"symbol"`
}

// NewClient creates a new database client
func NewClient(cfg config.DatabaseConfig) (*Client, error) {
	// Create connection string
	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.Host, cfg.User, cfg.Password, cfg.Name, cfg.Port, cfg.SSLMode)

	// Open database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	client := &Client{
		DB:     db,
		config: cfg,
	}

	return client, nil
}

// SetLogger sets the logger for the database client
func (c *Client) SetLogger(l *logger.Logger) {
	c.logger = l
}

// GetTokenForSymbol fetches the token for a given symbol
func (c *Client) GetTokenForSymbol(symbol string) (string, error) {
	var token string
	err := c.DB.QueryRow("SELECT token FROM stocks_list WHERE symbol = $1", symbol).Scan(&token)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("symbol not found: %s", symbol)
		}
		if c.logger != nil {
			c.logger.LogDatabaseError("GetTokenForSymbol", err)
		}
		return "", fmt.Errorf("error querying database: %w", err)
	}

	return token, nil
}

// GetSymbolToken fetches both symbol and token for a given symbol
func (c *Client) GetSymbolToken(symbol string) (*SymbolToken, error) {
	var symbolToken SymbolToken
	err := c.DB.QueryRow("SELECT symbol, token FROM stocks_list WHERE symbol = $1", symbol).Scan(&symbolToken.Symbol, &symbolToken.Token)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("symbol not found: %s", symbol)
		}
		if c.logger != nil {
			c.logger.LogDatabaseError("GetSymbolToken", err)
		}
		return nil, fmt.Errorf("error querying database: %w", err)
	}

	return &symbolToken, nil
}

// GetAllSymbols fetches all available symbols
func (c *Client) GetAllSymbols() ([]SymbolToken, error) {
	rows, err := c.DB.Query("SELECT symbol, token FROM stocks_list ORDER BY symbol")
	if err != nil {
		if c.logger != nil {
			c.logger.LogDatabaseError("GetAllSymbols", err)
		}
		return nil, fmt.Errorf("error querying database: %w", err)
	}
	defer rows.Close()

	var symbols []SymbolToken
	for rows.Next() {
		var st SymbolToken
		if err := rows.Scan(&st.Symbol, &st.Token); err != nil {
			if c.logger != nil {
				c.logger.LogDatabaseError("GetAllSymbols scan", err)
			}
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		symbols = append(symbols, st)
	}

	if err := rows.Err(); err != nil {
		if c.logger != nil {
			c.logger.LogDatabaseError("GetAllSymbols rows", err)
		}
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return symbols, nil
}

// SearchSymbols searches for symbols by partial match
func (c *Client) SearchSymbols(query string) ([]SymbolToken, error) {
	searchQuery := "%" + query + "%"
	rows, err := c.DB.Query("SELECT symbol, token FROM stocks_list WHERE symbol ILIKE $1 ORDER BY symbol LIMIT 50", searchQuery)
	if err != nil {
		if c.logger != nil {
			c.logger.LogDatabaseError("SearchSymbols", err)
		}
		return nil, fmt.Errorf("error querying database: %w", err)
	}
	defer rows.Close()

	var symbols []SymbolToken
	for rows.Next() {
		var st SymbolToken
		if err := rows.Scan(&st.Symbol, &st.Token); err != nil {
			if c.logger != nil {
				c.logger.LogDatabaseError("SearchSymbols scan", err)
			}
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		symbols = append(symbols, st)
	}

	if err := rows.Err(); err != nil {
		if c.logger != nil {
			c.logger.LogDatabaseError("SearchSymbols rows", err)
		}
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return symbols, nil
}

// Get Assets all assets in Portfolio
func (c *Client) GetHoldings(portfolioId string) ([]Asset, error) {
	rows, err := c.DB.Query("SELECT id, symbol, asset_type FROM assets WHERE portfolio_id = $1", portfolioId)
	if err != nil {
		if c.logger != nil {
			c.logger.LogDatabaseError("GetHoldings", err)
		}
		return nil, fmt.Errorf("error querying database: %w", err)
	}

	defer rows.Close()

	var assets []Asset
	for rows.Next() {
		var asset Asset
		if err := rows.Scan(&asset.ID, &asset.Symbol, &asset.Asset); err != nil {
			if c.logger != nil {
				c.logger.LogDatabaseError("GetHoldings scan", err)
			}
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		assets = append(assets, asset)
	}

	if err := rows.Err(); err != nil {
		if c.logger != nil {
			c.logger.LogDatabaseError("SearchSymbols rows", err)
		}
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return assets, nil

}

func (c *Client) GetTokens(symbols []string) (map[string]string, error) {
	if len(symbols) == 0 {
		return make(map[string]string), nil
	}

	// Build the query with proper placeholders
	placeholders := make([]string, len(symbols))
	args := make([]interface{}, len(symbols))
	for i, symbol := range symbols {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = symbol
	}

	query := fmt.Sprintf("SELECT symbol, token FROM stocks_list WHERE symbol IN (%s)", strings.Join(placeholders, ","))

	rows, err := c.DB.Query(query, args...)
	if err != nil {
		if c.logger != nil {
			c.logger.LogDatabaseError("GetTokens", err)
		}
		return nil, err
	}
	defer rows.Close()

	symbolToToken := make(map[string]string)
	for rows.Next() {
		var symbol, token string
		if err := rows.Scan(&symbol, &token); err != nil {
			if c.logger != nil {
				c.logger.LogDatabaseError("GetTokens scan", err)
			}
			return nil, err
		}
		symbolToToken[symbol] = token
	}

	if err := rows.Err(); err != nil {
		if c.logger != nil {
			c.logger.LogDatabaseError("GetTokens rows", err)
		}
		return nil, err
	}

	return symbolToToken, nil
}

// HealthCheck checks if the database is healthy
func (c *Client) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.DB.PingContext(ctx)
}

// Close closes the database connection
func (c *Client) Close() error {
	return c.DB.Close()
}

// GetMutualFundHoldings fetches the holdings for a given search_id
func (c *Client) GetMutualFundHoldings(search_id string) (map[string]interface{}, error) {
	url := "https://mf-openweb-search.dhan.co/SectorAllocation"
	response, err := http.Post(url, "application/json", bytes.NewBuffer([]byte(fmt.Sprintf(`
	{"entity_id":"DhanWeb","source":"W","token_id":"9c5688945773312281d7","data":{"scheme_isin":"%s"}}`, search_id))))
	if err != nil {
		return nil, fmt.Errorf("error getting mutual fund holdings: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	fmt.Println(string(body))

	var responseData map[string]interface{}
	if err := json.Unmarshal(body, &responseData); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	data, ok := responseData["data"]
	if !ok {
		return nil, fmt.Errorf("data not found in response")
	}

	return map[string]interface{}{
		"holdings": data,
	}, nil
}

// BulkUpdateCurrentPrices updates current_price for multiple assets in a single transaction
func (c *Client) BulkUpdateCurrentPrices(priceData map[string]float64) error {
	if len(priceData) == 0 {
		return nil
	}

	// Start a transaction
	tx, err := c.DB.Begin()
	if err != nil {
		if c.logger != nil {
			c.logger.LogDatabaseError("BulkUpdateCurrentPrices Begin", err)
		}
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare the bulk update statement
	stmt, err := tx.Prepare("UPDATE assets SET current_price = $1 WHERE symbol = $2")
	if err != nil {
		if c.logger != nil {
			c.logger.LogDatabaseError("BulkUpdateCurrentPrices Prepare", err)
		}
		return fmt.Errorf("error preparing statement: %w", err)
	}
	defer stmt.Close()

	// Execute updates for each symbol
	for symbol, price := range priceData {
		_, err := stmt.Exec(price, symbol)
		if err != nil {
			if c.logger != nil {
				c.logger.LogDatabaseError("BulkUpdateCurrentPrices Exec", err)
			}
			return fmt.Errorf("error updating price for symbol %s: %w", symbol, err)
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		if c.logger != nil {
			c.logger.LogDatabaseError("BulkUpdateCurrentPrices Commit", err)
		}
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}
