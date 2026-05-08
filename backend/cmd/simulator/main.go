// cmd/simulator — Thailand Election vote simulator (Phase 9).
//
// Usage examples:
//   go run ./cmd/simulator --mode=physical --rps=50 --duration=30s
//   go run ./cmd/simulator --mode=online   --rps=10 --duration=30s
//   go run ./cmd/simulator --verify
//   go run ./cmd/simulator --mode=physical --rps=100 --duration=60s --verify
//
// The simulator:
//   - Loads election master data (provinces, constituencies, candidates, parties) from DB
//   - For --mode=physical: POSTs signed vote payloads to POST /api/v1/votes
//   - For --mode=online: simulates the full voter auth + cast API flow
//   - For --verify: compares DB aggregate totals with Redis ZSET scores

package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/th-election/backend/internal/config"
)

func main() {
	// ── Flags ─────────────────────────────────────────────────────────────────
	mode     := flag.String("mode", "physical", "simulation mode: physical | online")
	rps      := flag.Int("rps", 50, "target requests per second")
	duration := flag.Duration("duration", 30*time.Second, "how long to run (e.g. 30s, 2m)")
	ballot   := flag.String("ballot", "all", "ballot type to simulate: all | CONSTITUENCY | PARTY_LIST | REFERENDUM")
	verify   := flag.Bool("verify", false, "compare DB totals vs Redis ZSETs after the run")
	logLevel := flag.String("log-level", "info", "log level: debug | info | warn | error")
	flag.Parse()

	// ── Logging ───────────────────────────────────────────────────────────────
	level, err := zerolog.ParseLevel(*logLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	// ── Config ────────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("load config")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ── DB pool ───────────────────────────────────────────────────────────────
	poolCfg, err := pgxpool.ParseConfig(cfg.DB.URL)
	if err != nil {
		log.Fatal().Err(err).Msg("parse DB URL")
	}
	poolCfg.MaxConns = 10
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("connect to DB")
	}
	defer pool.Close()

	// ── Redis (persistent) ────────────────────────────────────────────────────
	rdbOpts, err := redis.ParseURL(cfg.Redis.PersistentURL)
	if err != nil {
		log.Fatal().Err(err).Msg("parse Redis URL")
	}
	rdb := redis.NewClient(rdbOpts)
	defer rdb.Close()

	// ── Ed25519 simulator signing key ─────────────────────────────────────────
	privKeyB64 := viper.GetString("SIMULATOR_ED25519_PRIVATE_KEY")
	var privKey ed25519.PrivateKey
	if privKeyB64 == "" {
		log.Warn().Msg("SIMULATOR_ED25519_PRIVATE_KEY not set — generating ephemeral key (votes will fail sig check)")
		_, privKey, _ = ed25519.GenerateKey(nil)
	} else {
		raw, err := base64.StdEncoding.DecodeString(privKeyB64)
		if err != nil {
			log.Fatal().Err(err).Msg("decode SIMULATOR_ED25519_PRIVATE_KEY")
		}
		privKey = ed25519.PrivateKey(raw)
	}

	apiBase := fmt.Sprintf("http://localhost:%s/api/v1", cfg.Server.Port)

	// ── Load master data ──────────────────────────────────────────────────────
	log.Info().Msg("Loading election master data from DB…")
	data, err := loadElectionData(ctx, pool)
	if err != nil {
		log.Fatal().Err(err).Msg("load election data")
	}
	log.Info().
		Int("provinces", len(data.Provinces)).
		Int("parties", len(data.Parties)).
		Msg("Master data loaded")

	// ── Run simulation ────────────────────────────────────────────────────────
	var physOK, physFail, onlineOK, onlineFail int64

	runCtx, cancel := context.WithTimeout(ctx, *duration)
	defer cancel()

	switch *mode {
	case "physical":
		log.Info().
			Int("rps", *rps).
			Str("duration", duration.String()).
			Str("ballot", *ballot).
			Msg("Starting physical vote simulator")

		physOK, physFail = runPhysicalSimulator(runCtx, data, privKey, apiBase, *rps, *ballot)

		log.Info().
			Int64("succeeded", physOK).
			Int64("failed", physFail).
			Msg("Physical simulation complete")

	case "online":
		log.Info().
			Int("rps", *rps).
			Str("duration", duration.String()).
			Msg("Starting online vote simulator (requires OTP_DEV_MODE=true)")

		onlineOK, onlineFail = runOnlineSimulator(runCtx, apiBase, *rps)

		log.Info().
			Int64("succeeded", onlineOK).
			Int64("failed", onlineFail).
			Msg("Online simulation complete")

	default:
		log.Fatal().Str("mode", *mode).Msg("Unknown --mode (use 'physical' or 'online')")
	}

	// ── Optional verification ─────────────────────────────────────────────────
	allOK := true
	if *verify {
		log.Info().Msg("Verifying DB ↔ Redis consistency…")
		// Allow a couple of seconds for workers to flush
		time.Sleep(3 * time.Second)
		allOK = runVerify(ctx, pool, rdb)
	}

	printVerifySummary(allOK, physOK, physFail, onlineOK, onlineFail)

	if !allOK {
		os.Exit(1)
	}
}
