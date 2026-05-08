package worker

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/th-election/backend/internal/cache"
	vkafka "github.com/th-election/backend/internal/kafka"
	"github.com/th-election/backend/internal/realtime"
	db "github.com/th-election/backend/internal/db/sqlc"
)

// sentinelUUID is the zero UUID used as candidate_id for REFERENDUM rows
// in province_summaries (enables a NOT NULL composite PK without nullable columns).
var sentinelUUID = uuid.MustParse("00000000-0000-0000-0000-000000000000")

// AtomicUpdater processes a single VoteMessage in one database transaction,
// then propagates the result to Redis and Centrifugo.
//
// The critical guarantee: the DB transaction either commits fully or not at all.
// Redis / Centrifugo updates happen AFTER a successful commit — a partial Redis
// update is tolerable because the reconciliation job corrects drift.
type AtomicUpdater struct {
	pool      *pgxpool.Pool
	redis     *cache.Clients
	centrifugo *realtime.Client
}

// NewAtomicUpdater constructs an AtomicUpdater.
func NewAtomicUpdater(pool *pgxpool.Pool, redis *cache.Clients, centrifugo *realtime.Client) *AtomicUpdater {
	return &AtomicUpdater{pool: pool, redis: redis, centrifugo: centrifugo}
}

// Process applies msg to the database inside a serializable transaction,
// updates Redis ZSETs/HASHes, and broadcasts to Centrifugo.
func (u *AtomicUpdater) Process(ctx context.Context, msg *vkafka.VoteMessage) error {
	tx, err := u.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	q := db.New(tx)

	// ── Step 1: Insert the canonical vote event ───────────────────────────────
	params, err := buildInsertParams(msg)
	if err != nil {
		return fmt.Errorf("build insert params: %w", err)
	}
	if _, err = q.InsertVoteEvent(ctx, params); err != nil {
		return fmt.Errorf("insert vote event: %w", err)
	}

	// ── Step 2: Update the appropriate read model(s) ─────────────────────────
	switch msg.BallotType {
	case "CONSTITUENCY":
		if err = u.applyConstituency(ctx, q, msg); err != nil {
			return err
		}
	case "PARTY_LIST":
		if err = u.applyPartyList(ctx, q, msg); err != nil {
			return err
		}
	case "REFERENDUM":
		if err = u.applyReferendum(ctx, q, msg); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown ballot_type: %s", msg.BallotType)
	}

	// ── Step 3: Commit ────────────────────────────────────────────────────────
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	// ── Step 4: Post-commit side effects (best-effort) ───────────────────────
	u.updateRedis(ctx, msg)
	u.broadcast(ctx, msg)

	return nil
}

// ── DB helpers ────────────────────────────────────────────────────────────────

func buildInsertParams(msg *vkafka.VoteMessage) (db.InsertVoteEventParams, error) {
	var p db.InsertVoteEventParams

	p.BallotType = db.BallotType(msg.BallotType)
	p.Source = db.VoteSource(msg.Source)
	p.ProvinceID = msg.ProvinceID
	p.VoteCount = msg.VoteCount
	p.IdempotencyKey = msg.IdempotencyKey
	if msg.PayloadSignature != "" {
		p.PayloadSignature = &msg.PayloadSignature
	}

	// Anonymous token — required
	if msg.AnonymousToken != "" {
		tok, err := uuid.Parse(msg.AnonymousToken)
		if err != nil {
			return p, fmt.Errorf("parse anonymous_token: %w", err)
		}
		p.AnonymousToken = pgtype.UUID{Bytes: tok, Valid: true}
	}

	// Constituency ID (nullable)
	if msg.ConstituencyID != "" {
		cid, err := uuid.Parse(msg.ConstituencyID)
		if err != nil {
			return p, fmt.Errorf("parse constituency_id: %w", err)
		}
		p.ConstituencyID = pgtype.UUID{Bytes: cid, Valid: true}
	}

	// Candidate ID (nullable)
	if msg.CandidateID != "" {
		cid, err := uuid.Parse(msg.CandidateID)
		if err != nil {
			return p, fmt.Errorf("parse candidate_id: %w", err)
		}
		p.CandidateID = pgtype.UUID{Bytes: cid, Valid: true}
	}

	// Referendum vote (nullable)
	if msg.ReferendumVote != "" {
		p.ReferendumVote = db.NullReferendumVote{
			ReferendumVote: db.ReferendumVote(msg.ReferendumVote),
			Valid:          true,
		}
	}

	return p, nil
}

func (u *AtomicUpdater) applyConstituency(ctx context.Context, q *db.Queries, msg *vkafka.VoteMessage) error {
	if msg.ConstituencyID == "" || msg.CandidateID == "" {
		return fmt.Errorf("constituency ballot missing constituency_id or candidate_id")
	}

	constID, err := uuid.Parse(msg.ConstituencyID)
	if err != nil {
		return fmt.Errorf("parse constituency_id: %w", err)
	}
	candID, err := uuid.Parse(msg.CandidateID)
	if err != nil {
		return fmt.Errorf("parse candidate_id: %w", err)
	}

	online, physical := splitOnlinePhysical(msg)

	if err := q.UpsertConstituencySummary(ctx, db.UpsertConstituencySummaryParams{
		ConstituencyID: constID,
		CandidateID:    candID,
		TotalVotes:     int64(msg.VoteCount),
		OnlineVotes:    online,
		PhysicalVotes:  physical,
	}); err != nil {
		return fmt.Errorf("upsert constituency summary: %w", err)
	}

	if err := q.UpsertProvinceSummary(ctx, db.UpsertProvinceSummaryParams{
		ProvinceID:  msg.ProvinceID,
		BallotType:  db.BallotTypeCONSTITUENCY,
		CandidateID: candID,
		TotalVotes:  int64(msg.VoteCount),
	}); err != nil {
		return fmt.Errorf("upsert province summary (constituency): %w", err)
	}

	return nil
}

func (u *AtomicUpdater) applyPartyList(ctx context.Context, q *db.Queries, msg *vkafka.VoteMessage) error {
	if msg.CandidateID == "" {
		return fmt.Errorf("party_list ballot missing candidate_id (party UUID)")
	}

	partyID, err := uuid.Parse(msg.CandidateID)
	if err != nil {
		return fmt.Errorf("parse party uuid (candidate_id): %w", err)
	}

	online, physical := splitOnlinePhysical(msg)

	if err := q.UpsertPartyListNational(ctx, db.UpsertPartyListNationalParams{
		PartyID:       partyID,
		TotalVotes:    int64(msg.VoteCount),
		OnlineVotes:   online,
		PhysicalVotes: physical,
	}); err != nil {
		return fmt.Errorf("upsert party list national: %w", err)
	}

	return nil
}

func (u *AtomicUpdater) applyReferendum(ctx context.Context, q *db.Queries, msg *vkafka.VoteMessage) error {
	if msg.ReferendumVote == "" {
		return fmt.Errorf("referendum ballot missing referendum_vote")
	}

	var agree, disagree, abstain int64
	switch msg.ReferendumVote {
	case "AGREE":
		agree = int64(msg.VoteCount)
	case "DISAGREE":
		disagree = int64(msg.VoteCount)
	case "ABSTAIN":
		abstain = int64(msg.VoteCount)
	default:
		return fmt.Errorf("unknown referendum_vote: %s", msg.ReferendumVote)
	}

	if err := q.UpsertReferendumProvinceSummary(ctx, db.UpsertReferendumProvinceSummaryParams{
		ProvinceID:    msg.ProvinceID,
		AgreeVotes:    agree,
		DisagreeVotes: disagree,
		AbstainVotes:  abstain,
		TotalVotes:    int64(msg.VoteCount),
	}); err != nil {
		return fmt.Errorf("upsert referendum province summary: %w", err)
	}

	if err := q.UpsertProvinceSummary(ctx, db.UpsertProvinceSummaryParams{
		ProvinceID:  msg.ProvinceID,
		BallotType:  db.BallotTypeREFERENDUM,
		CandidateID: sentinelUUID, // no candidate for referendum rows
		TotalVotes:  int64(msg.VoteCount),
	}); err != nil {
		return fmt.Errorf("upsert province summary (referendum): %w", err)
	}

	return nil
}

// ── Redis side effects ────────────────────────────────────────────────────────

func (u *AtomicUpdater) updateRedis(ctx context.Context, msg *vkafka.VoteMessage) {
	votes := float64(msg.VoteCount)

	switch msg.BallotType {
	case "CONSTITUENCY":
		// Province-level constituency leaderboard (ZSet: member=candidate_id, score=votes)
		u.redis.Persistent.ZIncrBy(ctx,
			cache.KeyProvinceConstituency(msg.ProvinceID),
			votes,
			msg.CandidateID,
		)

	case "PARTY_LIST":
		// National party-list leaderboard (ZSet: member=party_id, score=votes)
		u.redis.Persistent.ZIncrBy(ctx,
			cache.KeyNationalPartyListLeaderboard,
			votes,
			msg.CandidateID, // CandidateID = party UUID for PARTY_LIST
		)

	case "REFERENDUM":
		field := redisReferendumField(msg.ReferendumVote)
		if field == "" {
			return
		}
		// Province-level referendum Hash
		u.redis.Persistent.HIncrBy(ctx, cache.KeyProvinceReferendum(msg.ProvinceID), field, int64(msg.VoteCount))
		u.redis.Persistent.HIncrBy(ctx, cache.KeyProvinceReferendum(msg.ProvinceID), "total", int64(msg.VoteCount))
		// National referendum Hash
		u.redis.Persistent.HIncrBy(ctx, cache.KeyNationalReferendum, field, int64(msg.VoteCount))
		u.redis.Persistent.HIncrBy(ctx, cache.KeyNationalReferendum, "total", int64(msg.VoteCount))
	}

	// Invalidate cached seat calculation so the next request recomputes.
	if msg.BallotType == "PARTY_LIST" {
		_ = u.redis.Cache.Del(ctx, cache.KeyCachePartySeats()).Err()
	}
}

func redisReferendumField(vote string) string {
	switch vote {
	case "AGREE":
		return "agree"
	case "DISAGREE":
		return "disagree"
	case "ABSTAIN":
		return "abstain"
	default:
		return ""
	}
}

// ── Centrifugo broadcast ──────────────────────────────────────────────────────

type updateEvent struct {
	BallotType   string `json:"ballot_type"`
	ProvinceID   int16  `json:"province_id,omitempty"`
	CandidateID  string `json:"candidate_id,omitempty"`
	VoteCount    int32  `json:"vote_count"`
	ReferendumVote string `json:"referendum_vote,omitempty"`
}

func (u *AtomicUpdater) broadcast(ctx context.Context, msg *vkafka.VoteMessage) {
	evt := updateEvent{
		BallotType:     msg.BallotType,
		ProvinceID:     msg.ProvinceID,
		CandidateID:    msg.CandidateID,
		VoteCount:      msg.VoteCount,
		ReferendumVote: msg.ReferendumVote,
	}

	switch msg.BallotType {
	case "CONSTITUENCY":
		u.centrifugo.Publish(ctx, realtime.ChannelProvince(msg.ProvinceID), evt)
		u.centrifugo.Publish(ctx, realtime.ChannelThailand, evt)
	case "PARTY_LIST":
		u.centrifugo.Publish(ctx, realtime.ChannelThailand, evt)
	case "REFERENDUM":
		u.centrifugo.Publish(ctx, realtime.ChannelReferendum, evt)
		u.centrifugo.Publish(ctx, realtime.ChannelProvince(msg.ProvinceID), evt)
	}
}

// ── Utility ───────────────────────────────────────────────────────────────────

func splitOnlinePhysical(msg *vkafka.VoteMessage) (online, physical int64) {
	if msg.Source == "online" {
		return int64(msg.VoteCount), 0
	}
	return 0, int64(msg.VoteCount)
}

// ProcessWithRetry calls Process and logs errors, retrying up to maxRetries.
// Used by consumers that want simple retry semantics without a DLQ.
func (u *AtomicUpdater) ProcessWithRetry(ctx context.Context, msg *vkafka.VoteMessage, maxRetries int) {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := u.Process(ctx, msg); err != nil {
			log.Error().Err(err).
				Int("attempt", attempt).
				Str("ballot_type", msg.BallotType).
				Str("idempotency_key", msg.IdempotencyKey).
				Msg("atomic updater: processing failed")
			if attempt == maxRetries {
				log.Error().
					Str("idempotency_key", msg.IdempotencyKey).
					Msg("atomic updater: max retries exceeded — message dropped")
			}
			continue
		}
		return
	}
}

// ApplyReadModels applies only the read model updates (summary UPSERTs, Redis, Centrifugo)
// without inserting a new vote_events row. Use this for online votes, where the API handler
// already inserted vote_events atomically with voter_rights_used.
func (u *AtomicUpdater) ApplyReadModels(ctx context.Context, msg *vkafka.VoteMessage) error {
	tx, err := u.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	q := db.New(tx)

	switch msg.BallotType {
	case "CONSTITUENCY":
		if err = u.applyConstituency(ctx, q, msg); err != nil {
			return err
		}
	case "PARTY_LIST":
		if err = u.applyPartyList(ctx, q, msg); err != nil {
			return err
		}
	case "REFERENDUM":
		if err = u.applyReferendum(ctx, q, msg); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown ballot_type: %s", msg.BallotType)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	u.updateRedis(ctx, msg)
	u.broadcast(ctx, msg)
	return nil
}

// ApplyReadModelsWithRetry retries ApplyReadModels up to maxRetries times.
func (u *AtomicUpdater) ApplyReadModelsWithRetry(ctx context.Context, msg *vkafka.VoteMessage, maxRetries int) {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := u.ApplyReadModels(ctx, msg); err != nil {
			log.Error().Err(err).
				Int("attempt", attempt).
				Str("ballot_type", msg.BallotType).
				Str("idempotency_key", msg.IdempotencyKey).
				Msg("atomic updater: apply read models failed")
			if attempt == maxRetries {
				log.Error().
					Str("idempotency_key", msg.IdempotencyKey).
					Msg("atomic updater: max retries exceeded for read model update — message dropped")
			}
			continue
		}
		return
	}
}

