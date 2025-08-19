package responses

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// SuccessResponse represents a successful API response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
}

// ErrorResponse represents an error API response
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Code    int    `json:"code"`
}

// LTPResponse represents the LTP data in a successful response
type LTPResponse struct {
	Symbol string  `json:"symbol"`
	Token  string  `json:"token"`
	LTP    float64 `json:"ltp"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

// SendSuccess sends a successful JSON response
func SendSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

// SendSuccessWithMessage sends a successful JSON response with a custom message
func SendSuccessWithMessage(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// SendError sends an error JSON response
func SendError(c *gin.Context, code int, message string) {
	c.JSON(code, ErrorResponse{
		Success: false,
		Error:   message,
		Code:    code,
	})
}

// SendLTPResponse sends a successful LTP response
func SendLTPResponse(c *gin.Context, symbol, token string, ltp float64) {
	response := LTPResponse{
		Symbol: symbol,
		Token:  token,
		LTP:    ltp,
	}
	SendSuccess(c, response)
}

// SendHealthResponse sends a health check response
func SendHealthResponse(c *gin.Context, status string, services map[string]string) {
	response := HealthResponse{
		Status:    status,
		Timestamp: time.Now().Format(time.RFC3339),
		Services:  services,
	}
	c.JSON(http.StatusOK, response)
}
