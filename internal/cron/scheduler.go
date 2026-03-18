package cron

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/robfig/cron/v3"
	"github.com/mike/fitassist/internal/service"
)

type Scheduler struct {
	cron    *cron.Cron
	syncSvc *service.SyncService
}

func NewScheduler(syncSvc *service.SyncService) *Scheduler {
	return &Scheduler{
		cron:    cron.New(),
		syncSvc: syncSvc,
	}
}

// Start begins the cron scheduler with the given sync interval in minutes.
func (s *Scheduler) Start(intervalMinutes int) error {
	spec := fmt.Sprintf("@every %dm", intervalMinutes)

	_, err := s.cron.AddFunc(spec, func() {
		slog.Info("cron: starting scheduled sync")
		s.syncSvc.SyncAll(context.Background())
	})
	if err != nil {
		return fmt.Errorf("adding cron job: %w", err)
	}

	s.cron.Start()
	slog.Info("cron scheduler started", "interval_minutes", intervalMinutes)
	return nil
}

// Stop gracefully stops the scheduler.
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	slog.Info("cron scheduler stopped")
}
