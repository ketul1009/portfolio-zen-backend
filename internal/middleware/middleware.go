package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"portfolio-zen-backend/internal/config"
	"portfolio-zen-backend/internal/logger"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// Recovery middleware for handling panics
func Recovery(log *logger.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			log.Error("Panic recovered: %s", err)
		} else {
			log.Error("Panic recovered: %v", recovered)
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Internal server error",
			"code":    500,
		})
	})
}

// CORS middleware for handling cross-origin requests
func CORS() gin.HandlerFunc {
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	config.ExposeHeaders = []string{"Content-Length"}
	config.AllowCredentials = true
	config.MaxAge = 12 * time.Hour

	return cors.New(config)
}

// RequestLogging middleware for logging all requests
func RequestLogging(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get status code
		statusCode := c.Writer.Status()

		// Log the request
		log.LogRequest(
			c.Request.Method,
			c.Request.URL.Path,
			statusCode,
			latency,
			c.ClientIP(),
			c.Request.UserAgent(),
		)
	}
}

// RateLimiting middleware for rate limiting requests
func RateLimiting(cfg config.RateLimitConfig, log *logger.Logger) gin.HandlerFunc {
	limiters := &sync.Map{}

	return func(c *gin.Context) {
		// Get client IP address
		ip := c.ClientIP()

		// Get or create rate limiter for this IP
		limiter, _ := limiters.LoadOrStore(ip, rate.NewLimiter(
			rate.Limit(float64(cfg.RequestsPerMinute)/60.0),
			cfg.Burst,
		))

		// Check if the request is allowed
		if !limiter.(*rate.Limiter).Allow() {
			// Rate limit exceeded
			c.Header("Retry-After", "60")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error":   fmt.Sprintf("rate limit exceeded for IP: %s", ip),
				"code":    429,
			})

			log.LogRateLimitExceeded(ip)
			c.Abort()
			return
		}

		// Request is allowed, continue to the next handler
		c.Next()
	}
}

// Authentication middleware for protecting routes
func Authentication() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Authorization header is required",
				"code":    401,
			})
			c.Abort()
			return
		}

		// Check if it's a Bearer token
		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid authorization header format",
				"code":    401,
			})
			c.Abort()
			return
		}

		// Extract token
		token := authHeader[7:]
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Token is required",
				"code":    401,
			})
			c.Abort()
			return
		}

		// TODO: Validate token here
		// For now, just pass through
		c.Set("token", token)
		c.Next()
	}
}

// RequestID middleware for adding unique request IDs
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID is already set
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		// Set request ID in context and headers
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// Timeout middleware for setting request timeouts
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		// Create a channel to signal completion
		done := make(chan bool, 1)

		go func() {
			c.Next()
			done <- true
		}()

		select {
		case <-done:
			// Request completed successfully
		case <-ctx.Done():
			// Request timed out
			c.Abort()
			c.JSON(http.StatusRequestTimeout, gin.H{
				"success": false,
				"error":   "Request timeout",
				"code":    408,
			})
		}
	}
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("req_%d_%d", time.Now().UnixNano(), time.Now().Unix())
}
