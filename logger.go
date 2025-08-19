package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger represents a structured logger
type Logger struct {
	logger *log.Logger
}

// NewLogger creates a new logger
func NewLogger() *Logger {
	return &Logger{
		logger: log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile),
	}
}

// LogRequest logs information about an incoming request
func (l *Logger) LogRequest(c *gin.Context) {
	start := time.Now()

	// Process request
	c.Next()

	// Calculate latency
	latency := time.Since(start)

	// Get status code
	statusCode := c.Writer.Status()

	// Log the request
	l.logger.Printf("[REQUEST] %s %s | %d | %v | %s | %s",
		c.Request.Method,
		c.Request.URL.Path,
		statusCode,
		latency,
		c.ClientIP(),
		c.Request.UserAgent())
}

// LogError logs an error
func (l *Logger) LogError(format string, v ...interface{}) {
	l.logger.Printf("[ERROR] "+format, v...)
}

// LogInfo logs informational messages
func (l *Logger) LogInfo(format string, v ...interface{}) {
	l.logger.Printf("[INFO] "+format, v...)
}

// LogLTPRequest logs information about an LTP request
func (l *Logger) LogLTPRequest(symbol, token string, ltp float64, latency time.Duration) {
	l.logger.Printf("[LTP] Symbol: %s | Token: %s | LTP: %.2f | Latency: %v",
		symbol,
		token,
		ltp,
		latency)
}

// LogLTPError logs an error in fetching LTP
func (l *Logger) LogLTPError(symbol, token string, err error) {
	l.logger.Printf("[LTP_ERROR] Symbol: %s | Token: %s | Error: %v",
		symbol,
		token,
		err)
}

// LogRateLimitExceeded logs when a rate limit is exceeded
func (l *Logger) LogRateLimitExceeded(ip string) {
	l.logger.Printf("[RATE_LIMIT] IP: %s exceeded rate limit", ip)
}
