// Package scheduler provides cron-based job scheduling.
package scheduler

import (
	"context"
	"fmt"
	"sync"

	"github.com/robfig/cron/v3"
	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
)

// CronScheduler implements agent.Scheduler using robfig/cron.
type CronScheduler struct {
	cron   *cron.Cron
	agent  agent.Agent
	cfg    config.SchedulerConfig
	mu     sync.Mutex
	ids    map[string]cron.EntryID
}

// New creates a scheduler. If cfg.Enabled is false, Start/AddJob are no-ops.
func New(ag agent.Agent, cfg config.SchedulerConfig) *CronScheduler {
	return &CronScheduler{
		cron:  cron.New(),
		agent: ag,
		cfg:   cfg,
		ids:   make(map[string]cron.EntryID),
	}
}

// Start begins the cron loop.
func (s *CronScheduler) Start(ctx context.Context) error {
	if !s.cfg.Enabled {
		return nil
	}
	s.cron.Start()
	go func() {
		<-ctx.Done()
		s.cron.Stop()
	}()
	return nil
}

// Stop gracefully shuts down the scheduler.
func (s *CronScheduler) Stop(ctx context.Context) error {
	s.mu.Lock()
	stopCtx := s.cron.Stop()
	s.mu.Unlock()
	select {
	case <-stopCtx.Done():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// AddJob registers a cron job. Spec uses standard 5-field cron format (minute hour day month weekday).
func (s *CronScheduler) AddJob(spec string, job agent.Job) error {
	if !s.cfg.Enabled {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.ids[job.Name()]; ok {
		return fmt.Errorf("job %q already registered", job.Name())
	}
	id, err := s.cron.AddFunc(spec, func() {
		job.Run(context.Background())
	})
	if err != nil {
		return fmt.Errorf("add job: %w", err)
	}
	s.ids[job.Name()] = id
	return nil
}

// Ensure CronScheduler implements agent.Scheduler.
var _ agent.Scheduler = (*CronScheduler)(nil)
