package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Test with default values
	cfg, err := Load()
	if err == nil {
		t.Error("Expected error for missing required environment variables")
	}

	// Set required environment variables
	os.Setenv("DB_USER", "testuser")
	os.Setenv("DB_PASSWORD", "testpass")
	os.Setenv("DB_NAME", "testdb")
	os.Setenv("AONE_USERNAME", "testuser")
	os.Setenv("AONE_PASSWORD", "testpass")
	os.Setenv("AONE_API_KEY", "testkey")
	os.Setenv("AONE_TOKEN", "testtoken")

	// Test loading configuration
	cfg, err = Load()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify default values
	if cfg.Server.Port != "8000" {
		t.Errorf("Expected default port 8000, got %s", cfg.Server.Port)
	}

	if cfg.Database.Host != "localhost" {
		t.Errorf("Expected default database host localhost, got %s", cfg.Database.Host)
	}

	if cfg.RateLimit.RequestsPerMinute != 60 {
		t.Errorf("Expected default rate limit 60, got %d", cfg.RateLimit.RequestsPerMinute)
	}

	// Test custom values
	os.Setenv("PORT", "9000")
	os.Setenv("DB_HOST", "testhost")
	os.Setenv("RATE_LIMIT_RPM", "120")

	cfg, err = Load()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if cfg.Server.Port != "9000" {
		t.Errorf("Expected custom port 9000, got %s", cfg.Server.Port)
	}

	if cfg.Database.Host != "testhost" {
		t.Errorf("Expected custom database host testhost, got %s", cfg.Database.Host)
	}

	if cfg.RateLimit.RequestsPerMinute != 120 {
		t.Errorf("Expected custom rate limit 120, got %d", cfg.RateLimit.RequestsPerMinute)
	}

	// Test duration parsing
	os.Setenv("SERVER_READ_TIMEOUT", "45s")
	cfg, err = Load()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if cfg.Server.ReadTimeout != 45*time.Second {
		t.Errorf("Expected custom read timeout 45s, got %v", cfg.Server.ReadTimeout)
	}

	// Clean up
	os.Unsetenv("DB_USER")
	os.Unsetenv("DB_PASSWORD")
	os.Unsetenv("DB_NAME")
	os.Unsetenv("AONE_USERNAME")
	os.Unsetenv("AONE_PASSWORD")
	os.Unsetenv("AONE_API_KEY")
	os.Unsetenv("AONE_TOKEN")
	os.Unsetenv("PORT")
	os.Unsetenv("DB_HOST")
	os.Unsetenv("RATE_LIMIT_RPM")
	os.Unsetenv("SERVER_READ_TIMEOUT")
}
