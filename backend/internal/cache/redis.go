package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/th-election/backend/internal/config"
)

// Clients holds the three separate Redis instances (PRD §2.4.2).
type Clients struct {
	// Persistent: leaderboard ZSETs, idempotency keys, vote locks, OTP hashes.
	// maxmemory-policy = noeviction
	Persistent *redis.Client

	// Cache: rate-limit counters, province/candidate JSON, seat calculation cache.
	// maxmemory-policy = allkeys-lru
	Cache *redis.Client

	// Asynq: job queue backing store — owned by the asynq library.
	// maxmemory-policy = noeviction
	Asynq *redis.Client
}

func NewClients(cfg *config.Config) (*Clients, error) {
	persistent, err := newClient(cfg.Redis.PersistentURL, "redis-persistent")
	if err != nil {
		return nil, err
	}
	cache, err := newClient(cfg.Redis.CacheURL, "redis-cache")
	if err != nil {
		return nil, err
	}
	asynq, err := newClient(cfg.Redis.AsynqURL, "redis-asynq")
	if err != nil {
		return nil, err
	}
	return &Clients{Persistent: persistent, Cache: cache, Asynq: asynq}, nil
}

func (c *Clients) Close() {
	_ = c.Persistent.Close()
	_ = c.Cache.Close()
	_ = c.Asynq.Close()
}

func newClient(url, name string) (*redis.Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse %s url: %w", name, err)
	}
	opts.DialTimeout = 3 * time.Second
	opts.ReadTimeout = 2 * time.Second
	opts.WriteTimeout = 2 * time.Second
	opts.PoolSize = 20
	opts.MinIdleConns = 2

	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping %s: %w", name, err)
	}
	return client, nil
}

// ─── Key constructors (PRD §6.4) ────────────────────────────────────────────

const (
	KeyNationalConstituencyLeaderboard = "election:national:constituency:leaderboard"
	KeyNationalPartyListLeaderboard    = "election:national:partylist:leaderboard"
	KeyNationalReferendum              = "election:national:referendum"

	KeyTTLIdempotency  = 24 * time.Hour
	KeyTTLVoteLock     = 10 * time.Second
	KeyTTLOTP          = 5 * time.Minute
	KeyTTLOTPAttempts  = 15 * time.Minute
	KeyTTLProvinces    = time.Hour
	KeyTTLCandidates   = time.Hour
	KeyTTLPartySeats   = 30 * time.Second
	KeyTTLBallot       = time.Hour
)

func KeyProvinceConstituency(provinceID int16) string {
	return fmt.Sprintf("election:province:%d:constituency", provinceID)
}

func KeyProvincePartyList(provinceID int16) string {
	return fmt.Sprintf("election:province:%d:partylist", provinceID)
}

func KeyProvinceReferendum(provinceID int16) string {
	return fmt.Sprintf("election:province:%d:referendum", provinceID)
}

func KeyIdempotency(key string) string {
	return "idempotency:" + key
}

func KeyVoteLock(sessionID string) string {
	return "vote_lock:" + sessionID
}

func KeyOTP(nationalIDHash string) string {
	return "otp:" + nationalIDHash
}

func KeyOTPAttempts(ip string) string {
	return "otp_attempts:" + ip
}

func KeyCacheProvincesList() string {
	return "cache:provinces:list"
}

func KeyCacheCandidatesList() string {
	return "cache:candidates:list"
}

func KeyCachePartySeats() string {
	return "cache:parties:seats"
}

func KeyCacheBallot(provinceID int16, ballotType string) string {
	return fmt.Sprintf("cache:ballot:%d:%s", provinceID, ballotType)
}
