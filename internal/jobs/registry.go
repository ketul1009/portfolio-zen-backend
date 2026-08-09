package jobs

import (
	"portfolio-zen-backend/internal/database"
	"portfolio-zen-backend/internal/logger"
	"portfolio-zen-backend/internal/scheduler"
	"portfolio-zen-backend/internal/services"

	"github.com/go-redis/redis/v8"
)

// Dependencies holds all the dependencies required by the jobs
type ScheduledJobDependencies struct {
	Logger      *logger.Logger
	DB          *database.Client
	Broker      *services.BrokerService
	RedisClient *redis.Client
}

// JobRegistry maps job names to their execution functions
var JobRegistry = map[string]func(deps *ScheduledJobDependencies) error{
	"update_portfolio": UpdatePortfolio,
	"process_sips":     ProcessSips,
}

// RegisterJobs registers all the cron jobs
func RegisterJobs(s *scheduler.Scheduler, deps ScheduledJobDependencies) {
	// Sample Job: Log every minute
	// _, err := s.AddJob("* * * * *", func() {
	// 	deps.Logger.Info("Cron job executed at %s", time.Now().Format(time.RFC3339))
	// 	fmt.Println("Cron job executed from internal/jobs")
	// })
	// if err != nil {
	// 	deps.Logger.Error("Error adding sample cron job: %v", err)
	// }

	_, err := s.AddJob("0 17 * * *", func() {
		if err := UpdatePortfolio(&deps); err != nil {
			deps.Logger.Error("[Cron] [RegisterJobs] Error executing UpdatePortfolio: %v", err)
		}
	})
	if err != nil {
		deps.Logger.Error("[Cron] [RegisterJobs] Error adding UpdatePortfolio cron job: %v", err)
	}

	_, err = s.AddJob("0 17 * * *", func() {
		if err := ProcessSips(&deps); err != nil {
			deps.Logger.Error("[Cron] [RegisterJobs] Error executing ProcessSips: %v", err)
		}
	})
	if err != nil {
		deps.Logger.Error("[Cron] [RegisterJobs] Error adding ProcessSips cron job: %v", err)
	}
}
