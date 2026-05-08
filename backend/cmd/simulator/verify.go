package main

// verify.go — compares PostgreSQL aggregate totals against Redis ZSET scores.
// Drifts > 0.01% are flagged as errors (matches the reconciliation worker threshold).

import (
	"context"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

const driftThresholdPct = 0.01 // 0.01%

// redisVerifier checks that Redis ZSETs match DB aggregates.
type redisVerifier struct {
	rdb *redis.Client
	db  interface {
		QueryRow(ctx context.Context, sql string, args ...any) interface{ Scan(dest ...any) error }
	}
}

// verifyResult holds the comparison outcome for one ballot type.
type verifyResult struct {
	BallotType string
	DBTotal    int64
	RedisTotal int64
	DriftPct   float64
	OK         bool
}

// runVerify performs the DB vs Redis comparison and prints a summary.
// Returns true if all comparisons pass within the drift threshold.
func runVerify(ctx context.Context, pgPool *pgxpool.Pool, rdb *redis.Client) bool {
	checks := []struct {
		ballotType string
		dbSQL      string
		redisKey   string
	}{
		{
			ballotType: "CONSTITUENCY",
			dbSQL: `SELECT COALESCE(SUM(vote_count), 0)
			        FROM vote_events
			        WHERE ballot_type = 'CONSTITUENCY'`,
			redisKey: "leaderboard:constituency:national",
		},
		{
			ballotType: "PARTY_LIST",
			dbSQL: `SELECT COALESCE(SUM(vote_count), 0)
			        FROM vote_events
			        WHERE ballot_type = 'PARTY_LIST'`,
			redisKey: "leaderboard:party_list:national",
		},
		{
			ballotType: "REFERENDUM",
			dbSQL: `SELECT COALESCE(SUM(vote_count), 0)
			        FROM vote_events
			        WHERE ballot_type = 'REFERENDUM'`,
			redisKey: "leaderboard:referendum:national",
		},
	}

	allOK := true

	for _, c := range checks {
		result := verifyBallotType(ctx, pgPool, rdb, c.ballotType, c.dbSQL, c.redisKey)

		status := "✓ OK"
		if !result.OK {
			status = "✗ DRIFT"
			allOK = false
		}

		log.Info().
			Str("ballot_type", result.BallotType).
			Int64("db_total", result.DBTotal).
			Int64("redis_total", result.RedisTotal).
			Float64("drift_pct", result.DriftPct).
			Msg(status)
	}

	return allOK
}

func verifyBallotType(
	ctx context.Context,
	pool *pgxpool.Pool,
	rdb *redis.Client,
	ballotType, sql, redisKey string,
) verifyResult {
	result := verifyResult{BallotType: ballotType}

	// Query DB total
	row := pool.QueryRow(ctx, sql)
	if err := row.Scan(&result.DBTotal); err != nil {
		log.Error().Err(err).Str("ballot_type", ballotType).Msg("DB query failed")
		return result
	}

	// Query Redis — sum all scores in the ZSET
	entries, err := rdb.ZRangeWithScores(ctx, redisKey, 0, -1).Result()
	if err != nil {
		log.Warn().Err(err).Str("key", redisKey).Msg("Redis ZSET read failed (key may not exist yet)")
		result.RedisTotal = 0
	} else {
		for _, e := range entries {
			result.RedisTotal += int64(e.Score)
		}
	}

	// Calculate drift
	if result.DBTotal == 0 && result.RedisTotal == 0 {
		result.OK = true
		return result
	}

	maxVal := math.Max(float64(result.DBTotal), float64(result.RedisTotal))
	if maxVal > 0 {
		result.DriftPct = math.Abs(float64(result.DBTotal-result.RedisTotal)) / maxVal * 100
	}
	result.OK = result.DriftPct <= driftThresholdPct

	return result
}


// printVerifySummary prints an overall pass/fail report.
func printVerifySummary(allOK bool, physSucceeded, physFailed, onlineSucceeded, onlineFailed int64) {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════")
	fmt.Println("  Simulator Run Summary")
	fmt.Println("═══════════════════════════════════════")
	if physSucceeded > 0 || physFailed > 0 {
		fmt.Printf("  Physical  — OK: %d  |  Failed: %d\n", physSucceeded, physFailed)
	}
	if onlineSucceeded > 0 || onlineFailed > 0 {
		fmt.Printf("  Online    — OK: %d  |  Failed: %d\n", onlineSucceeded, onlineFailed)
	}
	fmt.Println("───────────────────────────────────────")
	if allOK {
		fmt.Println("  DB ↔ Redis verification: ✓ PASS")
	} else {
		fmt.Println("  DB ↔ Redis verification: ✗ DRIFT DETECTED")
	}
	fmt.Println("═══════════════════════════════════════")
	fmt.Println()
}
