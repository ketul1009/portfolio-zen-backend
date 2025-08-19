package logger

import (
	"log"
	"os"
	"time"

	"portfolio-zen-backend/internal/config"
)

// Logger represents a structured logger
type Logger struct {
	logger *log.Logger
	config config.LoggingConfig
}

// New creates a new logger instance
func New(cfg config.LoggingConfig) (*Logger, error) {
	return &Logger{
		logger: log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile),
		config: cfg,
	}, nil
}

// Info logs informational messages
func (l *Logger) Info(format string, v ...interface{}) {
	l.logger.Printf("[INFO] "+format, v...)
}

// Error logs error messages
func (l *Logger) Error(format string, v ...interface{}) {
	l.logger.Printf("[ERROR] "+format, v...)
}

// Debug logs debug messages
func (l *Logger) Debug(format string, v ...interface{}) {
	if l.config.Level == "debug" {
		l.logger.Printf("[DEBUG] "+format, v...)
	}
}

// Warn logs warning messages
func (l *Logger) Warn(format string, v ...interface{}) {
	l.logger.Printf("[WARN] "+format, v...)
}

// LogRequest logs information about an incoming request
func (l *Logger) LogRequest(method, path string, statusCode int, latency time.Duration, clientIP, userAgent string) {
	l.Info("Request: %s %s | Status: %d | Latency: %v | IP: %s | UA: %s",
		method, path, statusCode, latency, clientIP, userAgent)
}

// LogLTPRequest logs information about an LTP request
func (l *Logger) LogLTPRequest(symbol, token string, ltp float64, latency time.Duration) {
	l.Info("LTP Request: Symbol: %s | Token: %s | LTP: %.2f | Latency: %v",
		symbol, token, ltp, latency)
}

// LogLTPError logs an error in fetching LTP
func (l *Logger) LogLTPError(symbol, token string, err error) {
	l.Error("LTP Error: Symbol: %s | Token: %s | Error: %v", symbol, token, err)
}

// LogRateLimitExceeded logs when a rate limit is exceeded
func (l *Logger) LogRateLimitExceeded(ip string) {
	l.Warn("Rate limit exceeded for IP: %s", ip)
}

// LogDatabaseError logs database-related errors
func (l *Logger) LogDatabaseError(operation string, err error) {
	l.Error("Database error in %s: %v", operation, err)
}

// LogBrokerError logs broker-related errors
func (l *Logger) LogBrokerError(operation string, err error) {
	l.Error("Broker error in %s: %v", operation, err)
}

// LogStartup logs application startup information
func (l *Logger) LogStartup(port string) {
	l.Info("Server starting on port %s", port)
}

// LogShutdown logs application shutdown information
func (l *Logger) LogShutdown() {
	l.Info("Server shutting down gracefully")
}

// LogHealthCheck logs health check results
func (l *Logger) LogHealthCheck(status string, latency time.Duration) {
	l.Info("Health check: Status: %s | Latency: %v", status, latency)
}
