package schedule

import (
	"context"
	"log/slog"

	"github.com/robfig/cron/v3"
)

// Scheduler runs the calendar sync and student list refresh every 5 minutes,
// mirroring the node-cron job in index.ts.
type Scheduler struct {
	cron    *cron.Cron
	syncer  *Syncer
	updater *StudentListUpdater
}

func NewScheduler(syncer *Syncer, updater *StudentListUpdater) *Scheduler {
	return &Scheduler{
		cron:    cron.New(),
		syncer:  syncer,
		updater: updater,
	}
}

// Start registers the "*/5 * * * *" job and starts the cron scheduler.
func (s *Scheduler) Start(ctx context.Context) error {
	_, err := s.cron.AddFunc("*/5 * * * *", func() {
		if !s.syncer.InProgress() {
			if err := s.syncer.Run(ctx); err != nil {
				slog.Error("scheduler: sync failed", "error", err)
			}
		}
		if err := s.updater.Run(ctx, false); err != nil {
			slog.Error("scheduler: student list update failed", "error", err)
		}
	})
	if err != nil {
		return err
	}

	s.cron.Start()
	return nil
}

// Stop stops the cron scheduler and waits for running jobs to finish.
func (s *Scheduler) Stop() {
	<-s.cron.Stop().Done()
}
