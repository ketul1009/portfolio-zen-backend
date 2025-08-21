package main

import (
	"log"

	"portfolio-zen-backend/internal/app"
	"portfolio-zen-backend/internal/config"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create and run the application
	app := app.New(cfg)

	// Start the worker in a separate goroutine
	go worker()

	if err := app.Run(); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}
