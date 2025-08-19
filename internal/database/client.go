package database

import (
	"context"
	"database/sql"
	"fmt"
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
	err := c.DB.QueryRow("SELECT symbol, token FROM symbol_tokens WHERE symbol = $1", symbol).Scan(&symbolToken.Symbol, &symbolToken.Token)
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
	rows, err := c.DB.Query("SELECT symbol, token FROM symbol_tokens ORDER BY symbol")
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
	rows, err := c.DB.Query("SELECT symbol, token FROM symbol_tokens WHERE symbol ILIKE $1 ORDER BY symbol LIMIT 50", searchQuery)
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
