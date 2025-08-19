package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// DBClient wraps the database connection
type DBClient struct {
	DB *sql.DB
}

// SymbolToken represents a symbol and its corresponding token
type SymbolToken struct {
	Symbol string
	Token  string
}

// NewDBClient creates a new database client
func NewDBClient() (*DBClient, error) {
	// Load .env file
	godotenv.Load()

	// Get database connection details from environment variables
	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	port := os.Getenv("DB_PORT")

	// Create connection string
	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=require", host, user, password, dbname, port)

	// Open database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	// Test the connection
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	return &DBClient{
		DB: db,
	}, nil
}

// GetTokenForSymbol fetches the token for a given symbol
func (dbc *DBClient) GetTokenForSymbol(symbol string) (string, error) {
	var token string
	err := dbc.DB.QueryRow("SELECT token FROM stocks_list WHERE symbol = $1", symbol).Scan(&token)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("symbol not found: %s", symbol)
		}
		return "", fmt.Errorf("error querying database: %w", err)
	}

	return token, nil
}

// GetSymbolToken fetches both symbol and token for a given symbol
func (dbc *DBClient) GetSymbolToken(symbol string) (*SymbolToken, error) {
	var symbolToken SymbolToken
	err := dbc.DB.QueryRow("SELECT symbol, token FROM symbol_tokens WHERE symbol = $1", symbol).Scan(&symbolToken.Symbol, &symbolToken.Token)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("symbol not found: %s", symbol)
		}
		return nil, fmt.Errorf("error querying database: %w", err)
	}

	return &symbolToken, nil
}

// Close closes the database connection
func (dbc *DBClient) Close() error {
	return dbc.DB.Close()
}
