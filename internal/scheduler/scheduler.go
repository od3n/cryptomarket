package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Scheduler runs a function periodically, preventing overlapping executions.
type Scheduler struct {
	interval time.Duration
	fn       func(ctx context.Context) error
	logger   *slog.Logger
	running  sync.Mutex
}

// New creates a new Scheduler.
func New(interval time.Duration, fn func(ctx context.Context) error, logger *slog.Logger) *Scheduler {
	return &Scheduler{
		interval: interval,
		fn:       fn,
		logger:   logger,
	}
}

// Start begins the periodic execution. It blocks until the context is canceled.
func (s *Scheduler) Start(ctx context.Context) {
	s.logger.Info("scheduler started", slog.Duration("interval", s.interval))

	// Run immediately on start.
	s.execute(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler stopped")
			return
		case <-ticker.C:
			s.execute(ctx)
		}
	}
}

// execute runs the function, preventing overlapping calls.
func (s *Scheduler) execute(ctx context.Context) {
	if !s.running.TryLock() {
		s.logger.Warn("skipping cycle: previous cycle still running")
		return
	}
	defer s.running.Unlock()

	if err := s.fn(ctx); err != nil {
		s.logger.Error("cycle failed", slog.String("error", err.Error()))
	}
}
