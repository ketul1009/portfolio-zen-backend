package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Broker      BrokerConfig
	RateLimit   RateLimitConfig
	Logging     LoggingConfig
	SQS         SQSConfigs
	Environment string
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// BrokerConfig holds broker API configuration
type BrokerConfig struct {
	Username        string
	Password        string
	APIKey          string
	TOTPSecret      string
	CoinGeckoAPIKey string
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute int
	Burst             int
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string
	Format string
}

// SQSConfig holds AWS SQS configuration
type SQSConfig struct {
	Region          string
	QueueURL        string
	AccessKeyID     string
	SecretAccessKey string
}

// SQSConfigs holds SQS configurations for different environments
type SQSConfigs struct {
	Production SQSConfig
	Gamma      SQSConfig
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:            getEnv("PORT", "8000"),
			ReadTimeout:     getEnvAsDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout:    getEnvAsDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:     getEnvAsDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getEnvAsDuration("SERVER_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", ""),
			Password:        getEnv("DB_PASSWORD", ""),
			Name:            getEnv("DB_NAME", ""),
			SSLMode:         getEnv("DB_SSLMODE", "require"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Broker: BrokerConfig{
			Username:        getEnv("AONE_USERNAME", ""),
			Password:        getEnv("AONE_PASSWORD", ""),
			APIKey:          getEnv("AONE_API_KEY", ""),
			TOTPSecret:      getEnv("AONE_TOKEN", ""),
			CoinGeckoAPIKey: getEnv("COINGECKO_API_KEY", ""),
		},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: getEnvAsInt("RATE_LIMIT_RPM", 60),
			Burst:             getEnvAsInt("RATE_LIMIT_BURST", 10),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		SQS: SQSConfigs{
			Production: SQSConfig{
				Region:          getEnv("AWS_REGION_PROD", "us-east-1"),
				QueueURL:        getEnv("SQS_QUEUE_URL_PROD", ""),
				AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID_PROD", ""),
				SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY_PROD", ""),
			},
			Gamma: SQSConfig{
				Region:          getEnv("AWS_REGION_GAMMA", getEnv("AWS_REGION", "us-east-1")),
				QueueURL:        getEnv("SQS_QUEUE_URL_GAMMA", getEnv("SQS_QUEUE_URL", "")),
				AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID_GAMMA", getEnv("AWS_ACCESS_KEY_ID", "")),
				SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY_GAMMA", getEnv("AWS_SECRET_ACCESS_KEY", "")),
			},
		},
		Environment: getEnv("ENVIRONMENT", "gamma"),
	}

	// Validate required configuration
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// GetActiveSQSConfig returns the SQS configuration based on the current environment
func (c *Config) GetActiveSQSConfig() SQSConfig {
	if c.Environment == "PROD" {
		return c.SQS.Production
	}
	return c.SQS.Gamma
}

// validate checks if required configuration values are present
func (c *Config) validate() error {
	var missingVars []string

	if c.Database.User == "" {
		missingVars = append(missingVars, "DB_USER")
	}
	if c.Database.Password == "" {
		missingVars = append(missingVars, "DB_PASSWORD")
	}
	if c.Database.Name == "" {
		missingVars = append(missingVars, "DB_NAME")
	}
	if c.Broker.Username == "" {
		missingVars = append(missingVars, "AONE_USERNAME")
	}
	if c.Broker.Password == "" {
		missingVars = append(missingVars, "AONE_PASSWORD")
	}
	if c.Broker.APIKey == "" {
		missingVars = append(missingVars, "AONE_API_KEY")
	}
	if c.Broker.TOTPSecret == "" {
		missingVars = append(missingVars, "AONE_TOKEN")
	}

	if len(missingVars) > 0 {
		return fmt.Errorf("missing required environment variables: %v. Please set them using 'fly secrets set' command", missingVars)
	}
	return nil
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
