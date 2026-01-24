package jobs

import (
	"fmt"
	"time"

	"portfolio-zen-backend/internal/database"
	"portfolio-zen-backend/internal/logger"
	"portfolio-zen-backend/internal/scheduler"
	"portfolio-zen-backend/internal/services"
)

// Dependencies holds all the dependencies required by the jobs
type Dependencies struct {
	Logger *logger.Logger
	DB     *database.Client
	Broker *services.BrokerService
}

// RegisterJobs registers all the cron jobs
func RegisterJobs(s *scheduler.Scheduler, deps Dependencies) {
	// Sample Job: Log every minute
	_, err := s.AddJob("* * * * *", func() {
		deps.Logger.Info("Cron job executed at %s", time.Now().Format(time.RFC3339))
		fmt.Println("Cron job executed from internal/jobs")
	})
	if err != nil {
		deps.Logger.Error("Error adding sample cron job: %v", err)
	}

	_, err = s.AddJob("@every 10s", func() {
		deps.Logger.Info("Cron job executed at %s", time.Now().Format(time.RFC3339))
		fmt.Println("New job test")
	})
	if err != nil {
		deps.Logger.Error("Error adding sample cron job: %v", err)
	}
}
