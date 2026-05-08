package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/th-election/backend/internal/cache"
	db "github.com/th-election/backend/internal/db/sqlc"
	"github.com/th-election/backend/internal/domain/reporting"
)

// Task type constants — used by both the scheduler (periodic tasks) and the
// API handler (on-demand tasks). Must match the strings used in api/handlers/online_voting.go.
const (
	// Periodic/scheduled tasks
	TaskReconcileVotes   = "votes:reconcile"
	TaskRecalcPartySeats = "votes:recalc_party_seats"
	TaskCleanupSessions  = "voter:session_cleanup"
	TaskExportAuditLog   = "admin:audit_export"

	// On-demand tasks enqueued by the online voting API
	TaskOTPSend     = "voter:otp:send"
	TaskReceiptIssue = "voter:receipt:issue"
)

// OTPSendPayload is the asynq task payload for TaskOTPSend.
type OTPSendPayload struct {
	SessionID string `json:"session_id"`
	DevOTP    string `json:"dev_otp,omitempty"` // only in dev mode
}

// ReceiptIssuePayload is the asynq task payload for TaskReceiptIssue.
type ReceiptIssuePayload struct {
	AnonymousToken string `json:"anonymous_token"`
	BallotType     string `json:"ballot_type"`
}

// TaskHandlers groups all asynq task handler functions.
type TaskHandlers struct {
	pool    *pgxpool.Pool
	queries db.Querier
	redis   *cache.Clients
}

// NewTaskHandlers creates a TaskHandlers instance.
func NewTaskHandlers(pool *pgxpool.Pool, queries db.Querier, redis *cache.Clients) *TaskHandlers {
	return &TaskHandlers{pool: pool, queries: queries, redis: redis}
}

// Register wires all task types to the asynq ServeMux.
func (h *TaskHandlers) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskReconcileVotes, h.handleReconcileVotes)
	mux.HandleFunc(TaskRecalcPartySeats, h.handleRecalcPartySeats)
	mux.HandleFunc(TaskCleanupSessions, h.handleCleanupSessions)
	mux.HandleFunc(TaskExportAuditLog, h.handleExportAuditLog)
	mux.HandleFunc(TaskOTPSend, h.handleOTPSend)
	mux.HandleFunc(TaskReceiptIssue, h.handleReceiptIssue)
}

// ── Task: ReconcileVotes ──────────────────────────────────────────────────────
// Compares Redis ZSETs against PostgreSQL aggregate sums and logs any drift.

func (h *TaskHandlers) handleReconcileVotes(ctx context.Context, _ *asynq.Task) error {
	log.Info().Msg("reconcile: starting vote reconciliation")

	// PostgreSQL totals from the party-list national read model
	rows, err := h.queries.GetPartyListNational(ctx)
	if err != nil {
		return fmt.Errorf("reconcile: get party list national: %w", err)
	}
	var dbTotal int64
	for _, r := range rows {
		dbTotal += r.TotalVotes
	}

	// Redis ZSET totals
	zscores, err := h.redis.Persistent.ZRangeWithScores(ctx,
		cache.KeyNationalPartyListLeaderboard, 0, -1).Result()
	if err != nil {
		log.Warn().Err(err).Msg("reconcile: redis ZRANGE failed, skipping comparison")
		return nil
	}
	var redisTotal float64
	for _, z := range zscores {
		redisTotal += z.Score
	}

	drift := int64(redisTotal) - dbTotal
	if drift < 0 {
		drift = -drift
	}
	if dbTotal > 0 {
		pct := float64(drift) / float64(dbTotal) * 100
		if pct > 0.01 {
			log.Warn().
				Int64("db_total", dbTotal).
				Float64("redis_total", redisTotal).
				Float64("drift_pct", pct).
				Msg("reconcile: DRIFT DETECTED — redis vs postgres mismatch")
		} else {
			log.Info().
				Int64("db_total", dbTotal).
				Float64("redis_total", redisTotal).
				Msg("reconcile: within tolerance")
		}
	}
	return nil
}

// ── Task: RecalcPartySeats ────────────────────────────────────────────────────
// Runs the proportional 100-seat party-list allocation (PRD §1.3.3) and persists
// results to party_list_national.seat_count.

func (h *TaskHandlers) handleRecalcPartySeats(ctx context.Context, _ *asynq.Task) error {
	rows, err := h.queries.GetPartyListNational(ctx)
	if err != nil {
		return fmt.Errorf("recalc seats: get party list: %w", err)
	}

	input := make([]reporting.PartyVotes, len(rows))
	for i, r := range rows {
		input[i] = reporting.PartyVotes{
			PartyID:        r.PartyID.String(),
			PartyName:      r.PartyName,
			PartyShortName: r.PartyShortName,
			PartyColor:     r.PartyColor,
			TotalVotes:     r.TotalVotes,
		}
	}

	allocs, _ := reporting.CalculatePartyListSeats(input)

	for _, alloc := range allocs {
		partyID, err := uuid.Parse(alloc.PartyID)
		if err != nil {
			log.Warn().Err(err).Str("party_id", alloc.PartyID).Msg("recalc seats: skip invalid UUID")
			continue
		}
		if err := h.queries.SetPartyListSeats(ctx, db.SetPartyListSeatsParams{
			PartyID:   partyID,
			SeatCount: int16(alloc.TotalSeats),
		}); err != nil {
			log.Warn().Err(err).Str("party_id", alloc.PartyID).Msg("recalc seats: update failed")
		}
	}

	// Bust the cached seat calculation so the next API request recomputes live.
	_ = h.redis.Cache.Del(ctx, cache.KeyCachePartySeats()).Err()

	log.Info().Int("parties", len(allocs)).Msg("recalc seats: complete")
	return nil
}

// ── Task: CleanupSessions ─────────────────────────────────────────────────────
// Marks voter_sessions as 'expired' when their expiry time has passed.

func (h *TaskHandlers) handleCleanupSessions(ctx context.Context, _ *asynq.Task) error {
	tag, err := h.pool.Exec(ctx,
		`UPDATE voter_sessions
		 SET    status = 'expired'
		 WHERE  status IN ('otp_pending', 'authenticated', 'voting')
		   AND  expires_at < NOW()`,
	)
	if err != nil {
		return fmt.Errorf("cleanup sessions: %w", err)
	}
	if tag.RowsAffected() > 0 {
		log.Info().Int64("expired", tag.RowsAffected()).Msg("session cleanup: sessions expired")
	}
	return nil
}

// ── Task: ExportAuditLog ──────────────────────────────────────────────────────
// Portfolio stub: logs the count of audit records for the past 24 hours.
// In production this would stream to S3/GCS.

func (h *TaskHandlers) handleExportAuditLog(ctx context.Context, _ *asynq.Task) error {
	var count int64
	if err := h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_logs WHERE created_at >= NOW() - INTERVAL '24 hours'`,
	).Scan(&count); err != nil {
		return fmt.Errorf("audit export: count: %w", err)
	}
	log.Info().
		Int64("audit_records_24h", count).
		Str("exported_at", time.Now().UTC().Format(time.RFC3339)).
		Msg("audit export: daily audit log export (portfolio: logged to stdout)")
	return nil
}

// ── Task: OTPSend ─────────────────────────────────────────────────────────────
// In development: logs the OTP to stdout so the frontend can display it in a
// yellow dev banner.  In production this would call an SMS provider (Twilio/SNS).

func (h *TaskHandlers) handleOTPSend(_ context.Context, t *asynq.Task) error {
	var p OTPSendPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("otp send: unmarshal payload: %w", err)
	}
	if p.DevOTP != "" {
		log.Info().
			Str("session_id", p.SessionID).
			Str("otp", p.DevOTP).
			Msg("OTP (dev mode) ── display this in the UI yellow dev banner")
	} else {
		log.Info().
			Str("session_id", p.SessionID).
			Msg("OTP send: production — would dispatch via SMS provider")
	}
	return nil
}

// ── Task: ReceiptIssue ────────────────────────────────────────────────────────
// Computes the deterministic receipt hash and persists it to vote_receipts.
// The hash is SHA-256(anonymous_token + "|" + ballot_type) — the same formula
// used by the cast handler so the voter gets an immediate receipt without waiting
// for this task to complete.

func (h *TaskHandlers) handleReceiptIssue(ctx context.Context, t *asynq.Task) error {
	var p ReceiptIssuePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("receipt issue: unmarshal payload: %w", err)
	}

	anonToken, err := uuid.Parse(p.AnonymousToken)
	if err != nil {
		return fmt.Errorf("receipt issue: parse anonymous token: %w", err)
	}

	sum := sha256.Sum256([]byte(p.AnonymousToken + "|" + p.BallotType))
	receiptHash := hex.EncodeToString(sum[:])

	if _, err := h.queries.InsertVoteReceipt(ctx, db.InsertVoteReceiptParams{
		ReceiptHash:    receiptHash,
		AnonymousToken: anonToken,
		BallotType:     db.BallotType(p.BallotType),
	}); err != nil {
		// A duplicate insert can happen on task retry — warn and succeed.
		log.Warn().Err(err).
			Str("receipt_hash", receiptHash).
			Msg("receipt issue: insert may be duplicate (retry?)")
	}

	log.Info().
		Str("receipt_hash", receiptHash).
		Str("ballot_type", p.BallotType).
		Msg("receipt issued")
	return nil
}
