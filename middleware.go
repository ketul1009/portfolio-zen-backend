package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
)

// RateLimiter struct holds the rate limiters for each IP
type RateLimiter struct {
	limiters sync.Map // Map of IP address to *rate.Limiter
	rate     rate.Limit
	burst    int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	// Load .env file
	godotenv.Load()

	// Get rate limit configuration from environment variables
	// Default to 60 requests per minute (1 per second) with a burst of 10
	requestsPerMinute, err := strconv.Atoi(os.Getenv("RATE_LIMIT_RPM"))
	if err != nil {
		requestsPerMinute = 60 // Default to 60 requests per minute
	}

	burst, err := strconv.Atoi(os.Getenv("RATE_LIMIT_BURST"))
	if err != nil {
		burst = 10 // Default to 10 burst requests
	}

	// Convert requests per minute to requests per second
	rate := rate.Limit(float64(requestsPerMinute) / 60.0)

	return &RateLimiter{
		limiters: sync.Map{},
		rate:     rate,
		burst:    burst,
	}
}

// getLimiter returns the rate limiter for the given IP address
func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	// Check if we already have a limiter for this IP
	limiter, ok := rl.limiters.Load(ip)
	if !ok {
		// Create a new limiter for this IP
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters.Store(ip, limiter)
	}

	return limiter.(*rate.Limiter)
}

// RateLimitMiddleware returns a Gin middleware function for rate limiting
func (rl *RateLimiter) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client IP address
		ip := c.ClientIP()

		// Get the rate limiter for this IP
		limiter := rl.getLimiter(ip)

		// Check if the request is allowed
		if !limiter.Allow() {
			// Rate limit exceeded
			c.Header("Retry-After", "60") // Suggest retrying after 60 seconds
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": fmt.Sprintf("rate limit exceeded for IP: %s", ip),
			})

			// Log rate limit exceeded
			logger := NewLogger()
			logger.LogRateLimitExceeded(ip)

			c.Abort()
			return
		}

		// Request is allowed, continue to the next handler
		c.Next()
	}
}
