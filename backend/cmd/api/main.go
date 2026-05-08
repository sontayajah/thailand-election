package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/th-election/backend/internal/api"
	"github.com/th-election/backend/internal/api/handlers"
	"github.com/th-election/backend/internal/cache"
	"github.com/th-election/backend/internal/config"
	appdb "github.com/th-election/backend/internal/db"
	dbsqlc "github.com/th-election/backend/internal/db/sqlc"
	"github.com/th-election/backend/internal/domain/voting"
	vkafka "github.com/th-election/backend/internal/kafka"
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

	log.Info().Str("env", cfg.App.Env).Msg("booting API server")

	ctx := context.Background()

	// ── PostgreSQL pool ──────────────────────────────────────────────────────
	pool, err := appdb.NewPool(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to postgres")
	}
	defer pool.Close()
	log.Info().Msg("postgres connected")

	// ── Redis clients (3 instances) ──────────────────────────────────────────
	redisClients, err := cache.NewClients(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis")
	}
	defer redisClients.Close()
	log.Info().Msg("redis clients connected")

	// ── Circuit breaker around redis-persistent ──────────────────────────────
	cbClient := cache.NewCircuitBreakerClient(redisClients.Persistent, "redis-persistent")

	// ── sqlc queries ─────────────────────────────────────────────────────────
	queries := dbsqlc.New(pool)

	// ── Kafka producer ───────────────────────────────────────────────────────
	producer := vkafka.NewProducer(cfg.Kafka.Brokers)
	defer producer.Close()
	log.Info().Strs("brokers", cfg.Kafka.Brokers).Msg("kafka producer initialised")

	// ── RSA key pair for voter JWT ───────────────────────────────────────────
	// Keys are generated once with `make generate-keys` and referenced via
	// JWT_PRIVATE_KEY_PATH / JWT_PUBLIC_KEY_PATH in .env.
	privateKey, err := voting.LoadPrivateKey(cfg.JWT.PrivateKeyPath)
	if err != nil {
		log.Fatal().Err(err).Str("path", cfg.JWT.PrivateKeyPath).Msg("failed to load JWT private key")
	}
	publicKey, err := voting.LoadPublicKey(cfg.JWT.PublicKeyPath)
	if err != nil {
		log.Fatal().Err(err).Str("path", cfg.JWT.PublicKeyPath).Msg("failed to load JWT public key")
	}
	log.Info().Msg("voter JWT key pair loaded")

	// ── asynq client (for enqueuing on-demand tasks from the API) ───────────
	asynqOpts, err := asynq.ParseRedisURI(cfg.Redis.AsynqURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse asynq redis URL")
	}
	asynqCli := asynq.NewClient(asynqOpts)
	defer asynqCli.Close()
	log.Info().Msg("asynq client connected")

	// ── Online voting handler ────────────────────────────────────────────────
	ovHandler := handlers.NewOnlineVotingHandler(
		cfg, pool, queries, redisClients, producer, privateKey, asynqCli,
	)

	// ── Admin handler ────────────────────────────────────────────────────────
	adminHandler := handlers.NewAdminHandler(cfg, queries, producer, privateKey)

	// ── Gin router ───────────────────────────────────────────────────────────
	router := api.NewRouter(cfg, pool, queries, redisClients, cbClient, producer, ovHandler, adminHandler, publicKey)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.Timeout,
		WriteTimeout: cfg.Server.Timeout,
		IdleTimeout:  cfg.Server.Timeout * 2,
	}

	// ── Start server ─────────────────────────────────────────────────────────
	go func() {
		log.Info().Str("addr", srv.Addr).Msg("API server listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	// ── Graceful shutdown ────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down gracefully...")
	shutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Error().Err(err).Msg("forced shutdown")
	}
	log.Info().Msg("server exited cleanly")
}

// Ensure pgxpool is imported via the db package.
var _ *pgxpool.Pool
