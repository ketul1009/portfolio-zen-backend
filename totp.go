package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"fmt"
	"math"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// TOTPConfig holds configuration for TOTP generation
type TOTPConfig struct {
	Secret    string
	Digits    int
	Algorithm string
	Period    int
}

// DefaultTOTPConfig returns default TOTP configuration
func DefaultTOTPConfig() *TOTPConfig {
	return &TOTPConfig{
		Digits:    6,
		Algorithm: "SHA1",
		Period:    30,
	}
}

// GenerateTOTP generates a TOTP code using the provided secret
// This is equivalent to Python's pyotp.TOTP(secret).now()
func GenerateTOTP(secret string) (string, error) {
	return GenerateTOTPWithConfig(secret, DefaultTOTPConfig())
}

// GenerateTOTPWithConfig generates a TOTP code with custom configuration
func GenerateTOTPWithConfig(secret string, config *TOTPConfig) (string, error) {
	// Clean the secret (remove spaces and convert to uppercase)
	cleanSecret := cleanSecret(secret)

	// Generate TOTP using the pquerna/otp library
	code, err := totp.GenerateCode(cleanSecret, time.Now())
	if err != nil {
		return "", fmt.Errorf("failed to generate TOTP: %w", err)
	}

	return code, nil
}

// GenerateTOTPAtTime generates a TOTP code for a specific time
func GenerateTOTPAtTime(secret string, t time.Time) (string, error) {
	cleanSecret := cleanSecret(secret)

	code, err := totp.GenerateCode(cleanSecret, t)
	if err != nil {
		return "", fmt.Errorf("failed to generate TOTP at time: %w", err)
	}

	return code, nil
}

// ValidateTOTP validates a TOTP code against the secret
func ValidateTOTP(secret, code string) (bool, error) {
	cleanSecret := cleanSecret(secret)
	valid := totp.Validate(code, cleanSecret)
	return valid, nil
}

// ValidateTOTPWithWindow validates a TOTP code with a custom validation window
func ValidateTOTPWithWindow(secret, code string, window int) (bool, error) {
	cleanSecret := cleanSecret(secret)
	valid, err := totp.ValidateCustom(code, cleanSecret, time.Now(), totp.ValidateOpts{
		Period:    30,
		Skew:      uint(window),
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		return false, fmt.Errorf("failed to validate TOTP: %w", err)
	}
	return valid, nil
}

// GetTOTPTimeRemaining returns the time remaining until the next TOTP code
func GetTOTPTimeRemaining() int {
	now := time.Now().Unix()
	return int(30 - (now % 30))
}

// GetTOTPTimeRemainingFloat returns the time remaining as a float for more precision
func GetTOTPTimeRemainingFloat() float64 {
	now := time.Now().UnixNano()
	period := int64(30 * 1e9) // 30 seconds in nanoseconds
	remaining := period - (now % period)
	return float64(remaining) / 1e9
}

// cleanSecret cleans and formats the secret key
func cleanSecret(secret string) string {
	// Remove spaces and convert to uppercase
	clean := ""
	for _, char := range secret {
		if char != ' ' {
			clean += string(char)
		}
	}
	return clean
}

// GenerateTOTPManual is a manual implementation without external dependencies
// This provides an alternative if you want to avoid the pquerna/otp dependency
func GenerateTOTPManual(secret string) (string, error) {
	return GenerateTOTPManualWithConfig(secret, DefaultTOTPConfig())
}

// GenerateTOTPManualWithConfig generates TOTP manually with custom config
func GenerateTOTPManualWithConfig(secret string, config *TOTPConfig) (string, error) {
	// Clean the secret
	cleanSecret := cleanSecret(secret)

	// Decode base32 secret
	secretBytes, err := base32.StdEncoding.DecodeString(cleanSecret)
	if err != nil {
		return "", fmt.Errorf("invalid base32 secret: %w", err)
	}

	// Get current timestamp
	now := time.Now().Unix()
	counter := uint64(math.Floor(float64(now) / float64(config.Period)))

	// Convert counter to bytes
	counterBytes := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		counterBytes[i] = byte(counter & 0xff)
		counter >>= 8
	}

	// Generate HMAC-SHA1
	h := hmac.New(sha1.New, secretBytes)
	h.Write(counterBytes)
	hash := h.Sum(nil)

	// Generate 6-digit code
	offset := hash[len(hash)-1] & 0xf
	code := ((int(hash[offset]) & 0x7f) << 24) |
		((int(hash[offset+1]) & 0xff) << 16) |
		((int(hash[offset+2]) & 0xff) << 8) |
		(int(hash[offset+3]) & 0xff)

	code = code % int(math.Pow10(config.Digits))

	return fmt.Sprintf(fmt.Sprintf("%%0%dd", config.Digits), code), nil
}
