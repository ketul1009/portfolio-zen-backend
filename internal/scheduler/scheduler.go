package scheduler

import (
	"log"

	"github.com/robfig/cron/v3"
)

// Scheduler wraps the cron library to provide task scheduling capabilities
type Scheduler struct {
	cron *cron.Cron
}

// NewScheduler creates a new scheduler instance
func NewScheduler() *Scheduler {
	// Create a new cron scheduler with seconds precision if needed,
	// but standard cron usually does minutes.
	// robfig/cron/v3 standard is minute-based.
	// We can use cron.New() for standard parser.
	c := cron.New()
	return &Scheduler{
		cron: c,
	}
}

// AddJob adds a new job to the scheduler with the given cron spec
func (s *Scheduler) AddJob(spec string, job func()) (cron.EntryID, error) {
	return s.cron.AddFunc(spec, job)
}

// Start starts the scheduler in a non-blocking background goroutine
func (s *Scheduler) Start() {
	s.cron.Start()
	log.Println("Scheduler started")
}

// Stop stops the scheduler and waits for running jobs to complete
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	log.Println("Scheduler stopped")
}

// Inspect returns entries
func (s *Scheduler) GetEntries() []cron.Entry {
	return s.cron.Entries()
}
