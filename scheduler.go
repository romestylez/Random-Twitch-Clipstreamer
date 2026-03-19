package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler wraps robfig/cron and manages a single daily fetch job.
type Scheduler struct {
	mu      sync.Mutex
	cron    *cron.Cron
	entryID cron.EntryID
	logger  *log.Logger
	job     func()
}

// NewScheduler creates a scheduler but does not start any job yet.
func NewScheduler(logger *log.Logger, job func()) *Scheduler {
	return &Scheduler{
		logger: logger,
		job:    job,
	}
}

// Start initialises the cron engine and registers a job if cfg.ScheduleEnabled is true.
func (s *Scheduler) Start(cfg Config) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cron = cron.New()
	s.cron.Start()

	if cfg.ScheduleEnabled {
		s.registerLocked(cfg)
	}
}

// Reschedule removes the old job and registers a new one based on updated cfg.
func (s *Scheduler) Reschedule(cfg Config) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cron == nil {
		return
	}
	if s.entryID != 0 {
		s.cron.Remove(s.entryID)
		s.entryID = 0
	}
	if cfg.ScheduleEnabled {
		s.registerLocked(cfg)
	} else {
		s.logger.Println("⏰ Scheduler disabled.")
	}
}

// NextRun returns the next scheduled run time, or zero if none scheduled.
func (s *Scheduler) NextRun() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cron == nil || s.entryID == 0 {
		return time.Time{}
	}
	entry := s.cron.Entry(s.entryID)
	return entry.Next
}

func (s *Scheduler) registerLocked(cfg Config) {
	expr := fmt.Sprintf("%d %d * * *", cfg.ScheduleMinute, cfg.ScheduleHour)
	id, err := s.cron.AddFunc(expr, s.job)
	if err != nil {
		s.logger.Printf("❌ Failed to register schedule %q: %v", expr, err)
		return
	}
	s.entryID = id
	next := s.cron.Entry(id).Next
	s.logger.Printf("⏰ Scheduler active: daily at %02d:%02d (next run: %s)",
		cfg.ScheduleHour, cfg.ScheduleMinute, next.Format("2006-01-02 15:04:05"))
}
