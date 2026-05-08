package handlers

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/th-election/backend/internal/api/middleware"
	"github.com/th-election/backend/internal/cache"
	"github.com/th-election/backend/internal/config"
	db "github.com/th-election/backend/internal/db/sqlc"
	"github.com/th-election/backend/internal/domain/voting"
	vkafka "github.com/th-election/backend/internal/kafka"
)

// ── asynq task identifiers (must match worker/tasks.go) ──────────────────────

const (
	taskOTPSend     = "voter:otp:send"
	taskReceiptIssue = "voter:receipt:issue"
)

// otpSendPayload is the JSON payload for the taskOTPSend asynq task.
type otpSendPayload struct {
	SessionID string `json:"session_id"`
	DevOTP    string `json:"dev_otp,omitempty"` // only populated in dev mode
}

// receiptIssuePayload is the JSON payload for the taskReceiptIssue asynq task.
type receiptIssuePayload struct {
	AnonymousToken string `json:"anonymous_token"`
	BallotType     string `json:"ballot_type"`
}

// ── Handler ───────────────────────────────────────────────────────────────────

// OnlineVotingHandler handles all voter-facing online voting endpoints.
type OnlineVotingHandler struct {
	cfg        *config.Config
	pool       *pgxpool.Pool
	queries    db.Querier
	redis      *cache.Clients
	producer   *vkafka.Producer
	privateKey *rsa.PrivateKey
	asynqCli   *asynq.Client
}

// NewOnlineVotingHandler wires up the handler.
func NewOnlineVotingHandler(
	cfg *config.Config,
	pool *pgxpool.Pool,
	queries db.Querier,
	redis *cache.Clients,
	producer *vkafka.Producer,
	privateKey *rsa.PrivateKey,
	asynqCli *asynq.Client,
) *OnlineVotingHandler {
	return &OnlineVotingHandler{
		cfg:        cfg,
		pool:       pool,
		queries:    queries,
		redis:      redis,
		producer:   producer,
		privateKey: privateKey,
		asynqCli:   asynqCli,
	}
}

// ── POST /online-voting/auth/verify-id  (B-12) ───────────────────────────────

type verifyIDRequest struct {
	NationalID string `json:"national_id" binding:"required,len=13"`
}

// VerifyID hashes the national_id, checks the voter registry, calls the DOPA mock,
// and creates a voter_session in otp_pending state.
func (h *OnlineVotingHandler) VerifyID(c *gin.Context) {
	var req verifyIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// SHA-256(national_id + pepper) — we never store the plaintext ID.
	idHash := hashNationalID(req.NationalID, h.cfg.App.NationalIDPepper)

	voter, err := h.queries.GetVoterByNationalIDHash(c.Request.Context(), idHash)
	if err != nil {
		// Deliberately vague — don't reveal whether the ID exists.
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "voter not found or not eligible"})
		return
	}
	if !voter.IsEligible {
		c.JSON(http.StatusForbidden, gin.H{"error": "voter is not eligible to cast a ballot"})
		return
	}

	// Call DOPA mock to confirm identity.
	if err := callDOPA(c.Request.Context(), h.cfg.DOPA.BaseURL, req.NationalID); err != nil {
		log.Warn().Err(err).Msg("verify-id: dopa check failed")
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "identity verification failed"})
		return
	}

	// Parse client IP (falls back to 0.0.0.0 if unparseable).
	ip, err := netip.ParseAddr(c.ClientIP())
	if err != nil {
		ip = netip.MustParseAddr("0.0.0.0")
	}
	ua := c.GetHeader("User-Agent")
	var uaPtr *string
	if ua != "" {
		uaPtr = &ua
	}

	session, err := h.queries.InsertVoterSession(c.Request.Context(), db.InsertVoterSessionParams{
		VoterRegistryID: voter.ID,
		IpAddress:       ip,
		UserAgent:       uaPtr,
	})
	if err != nil {
		log.Error().Err(err).Msg("verify-id: insert voter session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id":      session.ID,
		"expires_in":      int(time.Until(session.ExpiresAt.Time).Seconds()),
		"phone_masked":    maskPhone(voter.RegisteredPhone),
		"province_id":     voter.ProvinceID,
		"constituency_id": voter.ConstituencyID,
	})
}

// ── POST /online-voting/auth/request-otp  (B-13) ─────────────────────────────

type requestOTPRequest struct {
	SessionID string `json:"session_id" binding:"required,uuid"`
}

// RequestOTP generates a 6-digit OTP, stores its SHA-256 hash in Redis, and
// enqueues the voter:otp:send task. In dev mode the OTP is also returned in the
// response body so the frontend can display it in a yellow dev banner.
func (h *OnlineVotingHandler) RequestOTP(c *gin.Context) {
	var req requestOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sessionID, _ := uuid.Parse(req.SessionID)
	session, err := h.queries.GetVoterSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "session not found"})
		return
	}
	if session.Status != db.VoterSessionStatusOtpPending {
		c.JSON(http.StatusConflict, gin.H{"error": "session is not in otp_pending state"})
		return
	}
	if session.ExpiresAt.Time.Before(time.Now()) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "session has expired"})
		return
	}

	otp, err := generateOTP()
	if err != nil {
		log.Error().Err(err).Msg("request-otp: generate OTP")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate OTP"})
		return
	}

	// Store hashed OTP — never the plaintext.
	otpKey := cache.KeyOTP(sessionID.String())
	if err := h.redis.Persistent.Set(
		c.Request.Context(), otpKey, hashOTP(otp), cache.KeyTTLOTP,
	).Err(); err != nil {
		log.Error().Err(err).Msg("request-otp: store OTP hash")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store OTP"})
		return
	}

	// Enqueue OTP delivery task (dev: logs OTP; prod: calls SMS provider).
	var devOTP string
	if h.cfg.App.OTPDevMode {
		devOTP = otp
	}
	taskPayload, _ := json.Marshal(otpSendPayload{SessionID: sessionID.String(), DevOTP: devOTP})
	if _, err := h.asynqCli.Enqueue(asynq.NewTask(taskOTPSend, taskPayload)); err != nil {
		log.Warn().Err(err).Msg("request-otp: enqueue task")
	}

	resp := gin.H{"expires_in": int(cache.KeyTTLOTP.Seconds())}
	if h.cfg.App.OTPDevMode {
		resp["dev_otp"] = otp // displayed in yellow dev banner on frontend
	}
	c.JSON(http.StatusAccepted, resp)
}

// ── POST /online-voting/auth/verify-otp  (B-14) ──────────────────────────────

type verifyOTPRequest struct {
	SessionID string `json:"session_id" binding:"required,uuid"`
	OTP       string `json:"otp" binding:"required,len=6"`
}

// VerifyOTP validates the OTP with a constant-time compare, then issues an anonymous
// RS256 JWT. The JWT contains NO PII — only an anonymous_token UUID and session metadata.
func (h *OnlineVotingHandler) VerifyOTP(c *gin.Context) {
	var req verifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sessionID, _ := uuid.Parse(req.SessionID)
	session, err := h.queries.GetVoterSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "session not found"})
		return
	}
	if session.Status != db.VoterSessionStatusOtpPending {
		c.JSON(http.StatusConflict, gin.H{"error": "session is not in otp_pending state"})
		return
	}
	if session.ExpiresAt.Time.Before(time.Now()) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "session has expired"})
		return
	}

	// Increment attempt counter first (fail-safe: lock out before check).
	attempts, err := h.queries.IncrementOTPAttempts(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	if attempts > 5 {
		_ = h.queries.UpdateVoterSessionStatus(c.Request.Context(), db.UpdateVoterSessionStatusParams{
			ID:     sessionID,
			Status: db.VoterSessionStatusExpired,
		})
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many OTP attempts — session locked"})
		return
	}

	// Retrieve stored OTP hash.
	otpKey := cache.KeyOTP(sessionID.String())
	storedHash, err := h.redis.Persistent.Get(c.Request.Context(), otpKey).Result()
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "OTP expired or not requested"})
		return
	}

	// Constant-time comparison (PRD §10.2).
	inputHash := hashOTP(req.OTP)
	if subtle.ConstantTimeCompare([]byte(inputHash), []byte(storedHash)) != 1 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid OTP"})
		return
	}

	// OTP is valid — consume it (one-time use).
	_ = h.redis.Persistent.Del(c.Request.Context(), otpKey)

	if err := h.queries.UpdateVoterSessionStatus(c.Request.Context(), db.UpdateVoterSessionStatusParams{
		ID:     sessionID,
		Status: db.VoterSessionStatusAuthenticated,
	}); err != nil {
		log.Error().Err(err).Msg("verify-otp: update session status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to activate session"})
		return
	}

	// Generate a fresh anonymous token — this UUID is the only link between
	// vote_events records and is NOT traceable back to voter identity.
	anonymousToken := uuid.New()

	token, expiresAt, err := voting.SignVoterJWT(
		h.privateKey,
		sessionID,
		anonymousToken,
		session.ProvinceID,
		session.ConstituencyID,
		h.cfg.JWT.VoterTTL,
	)
	if err != nil {
		log.Error().Err(err).Msg("verify-otp: sign jwt")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"expires_at": expiresAt.UTC().Format(time.RFC3339),
	})
}

// ── GET /online-voting/eligibility  (B-15)  [requires voter JWT] ─────────────

// Eligibility returns which ballot types this session has already cast and which remain.
func (h *OnlineVotingHandler) Eligibility(c *gin.Context) {
	sessionIDStr := c.GetString(middleware.ContextKeySessionID)
	sessionID, _ := uuid.Parse(sessionIDStr)

	used, err := h.queries.GetVoterRightsUsed(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load eligibility"})
		return
	}

	castSet := make(map[string]bool)
	for _, u := range used {
		castSet[string(u.BallotType)] = true
	}

	allBallots := []string{"CONSTITUENCY", "PARTY_LIST", "REFERENDUM"}
	remaining := make([]string, 0)
	for _, b := range allBallots {
		if !castSet[b] {
			remaining = append(remaining, b)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"ballots_cast":      castSet,
		"ballots_remaining": remaining,
	})
}

// ── GET /online-voting/ballot/:ballot_type  (B-16)  [requires voter JWT] ─────

// GetBallot returns the candidates / parties / referendum options for the voter's
// constituency. Results are cached in redis-cache for KeyTTLBallot.
func (h *OnlineVotingHandler) GetBallot(c *gin.Context) {
	ballotType := strings.ToUpper(c.Param("ballot_type"))
	constituencyIDStr := c.GetString(middleware.ContextKeyConstituencyID)

	// Province-level cache key (party-list is national but keyed per province so
	// the same TTL and invalidation logic apply uniformly).
	provinceIDRaw, _ := c.Get(middleware.ContextKeyProvinceID)
	provinceID, _ := provinceIDRaw.(int16)

	cacheKey := cache.KeyCacheBallot(provinceID, ballotType)
	if cached, err := h.redis.Cache.Get(c.Request.Context(), cacheKey).Bytes(); err == nil {
		c.Data(http.StatusOK, "application/json; charset=utf-8", cached)
		return
	}

	var result interface{}
	switch ballotType {
	case "CONSTITUENCY":
		constID, _ := uuid.Parse(constituencyIDStr)
		candidates, err := h.queries.ListCandidatesByConstituency(
			c.Request.Context(),
			pgtype.UUID{Bytes: constID, Valid: true},
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load candidates"})
			return
		}
		result = candidates

	case "PARTY_LIST":
		candidates, err := h.queries.ListPartyListCandidates(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load party list"})
			return
		}
		result = candidates

	case "REFERENDUM":
		result = []gin.H{
			{"value": "AGREE", "label_th": "เห็นด้วย", "label_en": "Agree"},
			{"value": "DISAGREE", "label_th": "ไม่เห็นด้วย", "label_en": "Disagree"},
			{"value": "ABSTAIN", "label_th": "ไม่แสดงมติ", "label_en": "Abstain"},
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ballot_type: must be CONSTITUENCY, PARTY_LIST or REFERENDUM"})
		return
	}

	data, _ := json.Marshal(result)
	_ = h.redis.Cache.Set(c.Request.Context(), cacheKey, data, cache.KeyTTLBallot)
	c.Data(http.StatusOK, "application/json; charset=utf-8", data)
}

// ── POST /online-voting/cast  (B-17)  [requires voter JWT] ───────────────────

type castVoteRequest struct {
	BallotType     string `json:"ballot_type" binding:"required,oneof=CONSTITUENCY PARTY_LIST REFERENDUM"`
	CandidateID    string `json:"candidate_id"`
	ReferendumVote string `json:"referendum_vote"`
	// Confirm must be explicitly set to true — a separate acknowledgement step.
	Confirm *bool `json:"confirm" binding:"required"`
}

// CastVote is the critical atomic operation:
//  1. Quick pre-check: ballot type not already cast.
//  2. Acquire distributed Redis lock (10s TTL).
//  3. BEGIN TX → InsertVoteEvent + InsertVoterRightsUsed → COMMIT.
//     (voter_rights_used unique constraint is the last-resort double-cast guard)
//  4. Publish VoteMessage to votes.online Kafka topic (triggers read model updates).
//  5. Enqueue voter:receipt:issue asynq task.
//  6. Return 202 + receipt_hash (computed deterministically so the voter can verify).
func (h *OnlineVotingHandler) CastVote(c *gin.Context) {
	var req castVoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Confirm == nil || !*req.Confirm {
		c.JSON(http.StatusBadRequest, gin.H{"error": "confirm must be true to cast your vote"})
		return
	}

	// Validate ballot-type-specific fields.
	switch req.BallotType {
	case "CONSTITUENCY", "PARTY_LIST":
		if req.CandidateID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "candidate_id is required for " + req.BallotType})
			return
		}
	case "REFERENDUM":
		switch req.ReferendumVote {
		case "AGREE", "DISAGREE", "ABSTAIN":
			// valid
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "referendum_vote must be AGREE, DISAGREE or ABSTAIN"})
			return
		}
	}

	// Extract JWT claims set by the VoterJWT middleware.
	sessionIDStr := c.GetString(middleware.ContextKeySessionID)
	anonTokenStr := c.GetString(middleware.ContextKeyAnonymousToken)
	provinceIDRaw, _ := c.Get(middleware.ContextKeyProvinceID)
	provinceID, _ := provinceIDRaw.(int16)
	constituencyIDStr := c.GetString(middleware.ContextKeyConstituencyID)

	sessionID, _ := uuid.Parse(sessionIDStr)
	anonToken, _ := uuid.Parse(anonTokenStr)

	// Quick pre-check to avoid lock acquisition when the ballot is obviously cast.
	alreadyCast, err := h.queries.HasVoterCastBallot(c.Request.Context(), db.HasVoterCastBallotParams{
		VoterSessionID: sessionID,
		BallotType:     db.BallotType(req.BallotType),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "eligibility check failed"})
		return
	}
	if alreadyCast {
		c.JSON(http.StatusConflict, gin.H{"error": "you have already cast your " + req.BallotType + " ballot"})
		return
	}

	// Acquire distributed vote lock — prevents concurrent casts for the same session.
	lockKey := cache.KeyVoteLock(sessionIDStr)
	acquired, err := h.redis.Persistent.SetNX(c.Request.Context(), lockKey, "1", cache.KeyTTLVoteLock).Result()
	if err != nil || !acquired {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "a vote is already being processed for this session"})
		return
	}
	// Release lock on exit (best-effort; TTL is the safety net).
	defer h.redis.Persistent.Del(context.Background(), lockKey) //nolint:errcheck

	// Build the vote event parameters.
	idempotencyKey := sessionIDStr + "|" + req.BallotType
	params := db.InsertVoteEventParams{
		BallotType:     db.BallotType(req.BallotType),
		Source:         db.VoteSourceOnline,
		ProvinceID:     provinceID,
		VoteCount:      1,
		AnonymousToken: pgtype.UUID{Bytes: anonToken, Valid: true},
		IdempotencyKey: idempotencyKey,
	}
	switch req.BallotType {
	case "CONSTITUENCY":
		constID, _ := uuid.Parse(constituencyIDStr)
		params.ConstituencyID = pgtype.UUID{Bytes: constID, Valid: true}
		candID, _ := uuid.Parse(req.CandidateID)
		params.CandidateID = pgtype.UUID{Bytes: candID, Valid: true}
	case "PARTY_LIST":
		// CandidateID carries the party UUID for PARTY_LIST votes (same convention as physical).
		partyID, _ := uuid.Parse(req.CandidateID)
		params.CandidateID = pgtype.UUID{Bytes: partyID, Valid: true}
	case "REFERENDUM":
		params.ReferendumVote = db.NullReferendumVote{
			ReferendumVote: db.ReferendumVote(req.ReferendumVote),
			Valid:          true,
		}
	}

	// ── Atomic DB transaction ─────────────────────────────────────────────────
	tx, err := h.pool.BeginTx(c.Request.Context(), pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		log.Error().Err(err).Msg("cast-vote: begin tx")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(context.Background())
		}
	}()

	q := db.New(tx)

	if _, err := q.InsertVoteEvent(c.Request.Context(), params); err != nil {
		log.Error().Err(err).Msg("cast-vote: insert vote event")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record vote"})
		return
	}

	if err := q.InsertVoterRightsUsed(c.Request.Context(), db.InsertVoterRightsUsedParams{
		VoterSessionID: sessionID,
		BallotType:     db.BallotType(req.BallotType),
	}); err != nil {
		// Unique constraint violation = ballot already cast (caught a race condition).
		log.Warn().Err(err).Msg("cast-vote: insert voter rights used (possible race)")
		c.JSON(http.StatusConflict, gin.H{"error": "ballot already cast"})
		return
	}

	if err := tx.Commit(c.Request.Context()); err != nil {
		log.Error().Err(err).Msg("cast-vote: commit")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit vote"})
		return
	}
	committed = true
	// ── End of atomic transaction ─────────────────────────────────────────────

	// Update session status to 'voting' (best-effort, not critical).
	_ = h.queries.UpdateVoterSessionStatus(context.Background(), db.UpdateVoterSessionStatusParams{
		ID:     sessionID,
		Status: db.VoterSessionStatusVoting,
	})

	// Publish to votes.online — the online_consumer will update the read models
	// (constituency_summaries, party_list_national, etc.), Redis ZSETs, and
	// Centrifugo. NOTE: InsertVoteEvent was already done above; the worker must
	// NOT insert it again (uses ApplyReadModels instead of Process).
	voteMsg := &vkafka.VoteMessage{
		BallotType:     req.BallotType,
		Source:         "online",
		ProvinceID:     provinceID,
		ConstituencyID: constituencyIDStr,
		CandidateID:    req.CandidateID,
		ReferendumVote: req.ReferendumVote,
		VoteCount:      1,
		AnonymousToken: anonTokenStr,
		IdempotencyKey: idempotencyKey,
	}
	if err := h.producer.Publish(c.Request.Context(), voteMsg); err != nil {
		log.Warn().Err(err).Msg("cast-vote: kafka publish failed (non-fatal — summaries may lag)")
	}

	// Compute deterministic receipt hash and enqueue the receipt DB write.
	receiptHash := computeReceiptHash(anonTokenStr, req.BallotType)
	taskPayload, _ := json.Marshal(receiptIssuePayload{
		AnonymousToken: anonTokenStr,
		BallotType:     req.BallotType,
	})
	if _, err := h.asynqCli.Enqueue(asynq.NewTask(taskReceiptIssue, taskPayload)); err != nil {
		log.Warn().Err(err).Msg("cast-vote: enqueue receipt task")
	}

	c.JSON(http.StatusAccepted, gin.H{
		"receipt_hash": receiptHash,
		"ballot_type":  req.BallotType,
		"status":       "recorded",
	})
}

// ── GET /online-voting/receipt/:hash  (B-18)  [public] ───────────────────────

// GetReceipt is the public receipt verification endpoint.
// It confirms that a vote was recorded without revealing the choice.
func (h *OnlineVotingHandler) GetReceipt(c *gin.Context) {
	hash := c.Param("hash")
	if hash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "receipt hash is required"})
		return
	}

	receipt, err := h.queries.GetVoteReceiptByHash(c.Request.Context(), hash)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "receipt not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"receipt_hash": receipt.ReceiptHash,
		"ballot_type":  receipt.BallotType,
		"issued_at":    receipt.IssuedAt.Time.UTC().Format(time.RFC3339),
		"verified":     true,
	})
}

// ── Utility functions ─────────────────────────────────────────────────────────

// hashNationalID returns the hex-encoded SHA-256(national_id + pepper).
func hashNationalID(nationalID, pepper string) string {
	sum := sha256.Sum256([]byte(nationalID + pepper))
	return hex.EncodeToString(sum[:])
}

// hashOTP returns the hex-encoded SHA-256 of the OTP string.
func hashOTP(otp string) string {
	sum := sha256.Sum256([]byte(otp))
	return hex.EncodeToString(sum[:])
}

// generateOTP returns a cryptographically random 6-digit string (left-padded with zeros).
func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// maskPhone returns the phone number with all but the last 4 digits replaced by '*'.
func maskPhone(phone string) string {
	if len(phone) < 4 {
		return strings.Repeat("*", len(phone))
	}
	return strings.Repeat("*", len(phone)-4) + phone[len(phone)-4:]
}

// computeReceiptHash returns SHA-256(anonymous_token + "|" + ballot_type) as hex.
// This is deterministic so the voter receives the hash immediately on cast
// while the async task persists it to the DB.
func computeReceiptHash(anonymousToken, ballotType string) string {
	sum := sha256.Sum256([]byte(anonymousToken + "|" + ballotType))
	return hex.EncodeToString(sum[:])
}

// callDOPA posts the national_id to the DOPA mock service and checks the response.
func callDOPA(ctx context.Context, baseURL, nationalID string) error {
	body, err := json.Marshal(map[string]string{"national_id": nationalID})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/verify", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("dopa request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dopa: unexpected status %d", resp.StatusCode)
	}

	var result struct {
		Valid bool `json:"valid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("dopa: decode response: %w", err)
	}
	if !result.Valid {
		return fmt.Errorf("dopa: identity verification returned valid=false")
	}
	return nil
}
