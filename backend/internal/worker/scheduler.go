package worker

import (
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
)

// NewScheduler creates an asynq periodic task scheduler.
// The scheduler enqueues tasks into the asynq server on the configured cron schedule.
// It uses the same Redis instance as the asynq server (redis-asynq).
func NewScheduler(redisURL string) (*asynq.Scheduler, error) {
	opts, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse asynq redis URL: %w", err)
	}

	scheduler := asynq.NewScheduler(opts, &asynq.SchedulerOpts{
		LogLevel: asynq.InfoLevel,
		PostEnqueueFunc: func(info *asynq.TaskInfo, err error) {
			if err != nil {
				log.Warn().Err(err).Str("task", info.Type).Msg("scheduler: enqueue failed")
			}
		},
	})

	// Recalculate party-list seats every 30 seconds.
	if _, err := scheduler.Register("@every 30s",
		asynq.NewTask(TaskRecalcPartySeats, nil),
		asynq.MaxRetry(2),
	); err != nil {
		return nil, fmt.Errorf("register recalc party seats: %w", err)
	}

	// Reconcile Redis vs PostgreSQL every 5 minutes.
	if _, err := scheduler.Register("@every 5m",
		asynq.NewTask(TaskReconcileVotes, nil),
		asynq.MaxRetry(1),
	); err != nil {
		return nil, fmt.Errorf("register reconcile votes: %w", err)
	}

	// Clean up expired voter sessions every minute.
	if _, err := scheduler.Register("@every 1m",
		asynq.NewTask(TaskCleanupSessions, nil),
		asynq.MaxRetry(3),
	); err != nil {
		return nil, fmt.Errorf("register cleanup sessions: %w", err)
	}

	// Daily audit log export at midnight UTC.
	if _, err := scheduler.Register("0 0 * * *",
		asynq.NewTask(TaskExportAuditLog, nil),
		asynq.MaxRetry(2),
	); err != nil {
		return nil, fmt.Errorf("register audit export: %w", err)
	}

	return scheduler, nil
}

// NewAsynqServer creates an asynq.Server connected to redis-asynq.
// Concurrency is set to 10 — enough for the 4 scheduled tasks with room to spare.
func NewAsynqServer(redisURL string) (*asynq.Server, error) {
	opts, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse asynq redis URL for server: %w", err)
	}

	srv := asynq.NewServer(opts, asynq.Config{
		Concurrency: 10,
		Queues: map[string]int{
			"default": 6,
			"critical": 4,
		},
	})

	return srv, nil
}
