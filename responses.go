package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SuccessResponse represents a successful API response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
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

// SendSuccess sends a successful JSON response
func SendSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
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
