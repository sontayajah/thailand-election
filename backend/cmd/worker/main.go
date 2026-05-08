package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/th-election/backend/internal/cache"
	"github.com/th-election/backend/internal/config"
	appdb "github.com/th-election/backend/internal/db"
	dbsqlc "github.com/th-election/backend/internal/db/sqlc"
	"github.com/th-election/backend/internal/realtime"
	"github.com/th-election/backend/internal/worker"
)

func main() {
	zerolog.TimeFieldFormat = time.RFC3339
	if os.Getenv("APP_ENV") != "production" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}
	log.Info().Str("env", cfg.App.Env).Msg("booting worker")

	// ── PostgreSQL pool ──────────────────────────────────────────────────────
	ctx := context.Background()
	pool, err := appdb.NewPool(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to postgres")
	}
	defer pool.Close()
	log.Info().Msg("postgres connected")

	// ── Redis clients ────────────────────────────────────────────────────────
	redisClients, err := cache.NewClients(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis")
	}
	defer redisClients.Close()
	log.Info().Msg("redis clients connected")

	// ── sqlc queries ─────────────────────────────────────────────────────────
	queries := dbsqlc.New(pool)

	// ── Centrifugo client ────────────────────────────────────────────────────
	centrifugoClient := realtime.NewClient(cfg)

	// ── Atomic updater (shared by all 4 consumers) ───────────────────────────
	updater := worker.NewAtomicUpdater(pool, redisClients, centrifugoClient)

	// ── Kafka consumers ──────────────────────────────────────────────────────
	constituencyConsumer := worker.NewConstituencyConsumer(cfg.Kafka.Brokers, updater)
	partyListConsumer := worker.NewPartyListConsumer(cfg.Kafka.Brokers, updater)
	referendumConsumer := worker.NewReferendumConsumer(cfg.Kafka.Brokers, updater)
	onlineConsumer := worker.NewOnlineConsumer(cfg.Kafka.Brokers, updater)

	consumerCtx, consumerCancel := context.WithCancel(ctx)
	go constituencyConsumer.Run(consumerCtx)
	go partyListConsumer.Run(consumerCtx)
	go referendumConsumer.Run(consumerCtx)
	go onlineConsumer.Run(consumerCtx)
	log.Info().Msg("kafka consumers started")

	// ── asynq task server ────────────────────────────────────────────────────
	asynqSrv, err := worker.NewAsynqServer(cfg.Redis.AsynqURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create asynq server")
	}

	taskHandlers := worker.NewTaskHandlers(pool, queries, redisClients)
	mux := asynq.NewServeMux()
	taskHandlers.Register(mux)

	go func() {
		if err := asynqSrv.Run(mux); err != nil {
			log.Fatal().Err(err).Msg("asynq server error")
		}
	}()
	log.Info().Msg("asynq task server started")

	// ── asynq scheduler ──────────────────────────────────────────────────────
	scheduler, err := worker.NewScheduler(cfg.Redis.AsynqURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create asynq scheduler")
	}
	if err := scheduler.Start(); err != nil {
		log.Fatal().Err(err).Msg("failed to start asynq scheduler")
	}
	log.Info().Msg("asynq scheduler started")

	// ── Graceful shutdown ────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("worker: shutting down gracefully...")

	// Stop Kafka consumers first
	consumerCancel()

	// Stop asynq scheduler (stops enqueuing new tasks)
	scheduler.Shutdown()

	// Stop asynq server (waits for running tasks to finish, up to 30s)
	asynqSrv.Shutdown()

	log.Info().Msg("worker: exited cleanly")
}
