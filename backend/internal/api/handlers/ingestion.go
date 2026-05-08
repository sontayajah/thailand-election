package handlers

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/th-election/backend/internal/api/middleware"
	"github.com/th-election/backend/internal/cache"
	"github.com/th-election/backend/internal/config"
	db "github.com/th-election/backend/internal/db/sqlc"
	vkafka "github.com/th-election/backend/internal/kafka"
)

// IngestionHandler handles POST /api/v1/votes (B-01: physical vote ingestion).
type IngestionHandler struct {
	cfg      *config.Config
	queries  db.Querier
	redis    *cache.Clients
	producer *vkafka.Producer
}

// NewIngestionHandler constructs an IngestionHandler.
func NewIngestionHandler(
	cfg *config.Config,
	queries db.Querier,
	redis *cache.Clients,
	producer *vkafka.Producer,
) *IngestionHandler {
	return &IngestionHandler{cfg: cfg, queries: queries, redis: redis, producer: producer}
}

// VoteRequest is the body accepted by POST /api/v1/votes.
type VoteRequest struct {
	BallotType       string `json:"ballot_type"        binding:"required,oneof=CONSTITUENCY PARTY_LIST REFERENDUM"`
	ProvinceID       int16  `json:"province_id"        binding:"required,min=1,max=77"`
	ConstituencyID   string `json:"constituency_id"`
	CandidateID      string `json:"candidate_id"`
	ReferendumVote   string `json:"referendum_vote"    binding:"omitempty,oneof=AGREE DISAGREE ABSTAIN"`
	VoteCount        int32  `json:"vote_count"         binding:"required,min=1,max=10000"`
	IdempotencyKey   string `json:"idempotency_key"    binding:"required,min=8,max=128"`
	PayloadSignature string `json:"payload_signature"  binding:"required"`
}

// IngestVote handles B-01: POST /api/v1/votes.
//   - Validates the request body.
//   - Checks Redis for duplicate idempotency key (NX EX 24h).
//   - Verifies the Ed25519 signature against the province's public key.
//   - Publishes the vote to the appropriate Kafka topic.
//   - Returns 202 Accepted in < 10ms (DB work happens asynchronously in workers).
func (h *IngestionHandler) IngestVote(c *gin.Context) {
	var req VoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Bad Request", err.Error(), middleware.GetTraceID(c)))
		return
	}

	// Ballot-type-specific field validation
	if req.BallotType == "CONSTITUENCY" && (req.ConstituencyID == "" || req.CandidateID == "") {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Bad Request",
			"constituency_id and candidate_id are required for CONSTITUENCY ballots",
			middleware.GetTraceID(c)))
		return
	}
	if req.BallotType == "PARTY_LIST" && req.CandidateID == "" {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Bad Request",
			"candidate_id (party UUID) is required for PARTY_LIST ballots",
			middleware.GetTraceID(c)))
		return
	}
	if req.BallotType == "REFERENDUM" && req.ReferendumVote == "" {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Bad Request",
			"referendum_vote is required for REFERENDUM ballots",
			middleware.GetTraceID(c)))
		return
	}

	ctx := c.Request.Context()

	// ── Idempotency check ────────────────────────────────────────────────────
	ikey := cache.KeyIdempotency(req.IdempotencyKey)
	set, err := h.redis.Persistent.SetNX(ctx, ikey, "1", cache.KeyTTLIdempotency).Result()
	if err != nil {
		log.Warn().Err(err).Str("idempotency_key", req.IdempotencyKey).Msg("ingestion: idempotency redis error")
		// Proceed on Redis failure — the DB unique constraint is the hard guard.
	}
	if !set && err == nil {
		// Key already existed → duplicate delivery; return success idempotently.
		c.JSON(http.StatusOK, gin.H{"status": "duplicate", "message": "vote already accepted"})
		return
	}

	// ── Ed25519 signature verification ───────────────────────────────────────
	keyRow, err := h.queries.GetProvinceKey(ctx, req.ProvinceID)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, errorResponse(422, "Unprocessable Entity",
			"no active signing key for province", middleware.GetTraceID(c)))
		return
	}

	pubKey, err := parseEd25519PublicKey(keyRow.PublicKeyPem)
	if err != nil {
		log.Error().Err(err).Int("province_id", int(req.ProvinceID)).Msg("ingestion: parse ed25519 public key")
		c.JSON(http.StatusInternalServerError, errorResponse(500, "Internal Server Error",
			"key parse error", middleware.GetTraceID(c)))
		return
	}

	msg := &vkafka.VoteMessage{
		BallotType:       req.BallotType,
		Source:           "physical",
		ProvinceID:       req.ProvinceID,
		ConstituencyID:   req.ConstituencyID,
		CandidateID:      req.CandidateID,
		ReferendumVote:   req.ReferendumVote,
		VoteCount:        req.VoteCount,
		AnonymousToken:   uuid.New().String(),
		IdempotencyKey:   req.IdempotencyKey,
		PayloadSignature: req.PayloadSignature,
	}

	sigBytes, err := base64.StdEncoding.DecodeString(req.PayloadSignature)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Bad Request",
			"payload_signature must be standard-base64-encoded", middleware.GetTraceID(c)))
		return
	}

	if !ed25519.Verify(pubKey, msg.SignedBytes(), sigBytes) {
		c.JSON(http.StatusUnauthorized, errorResponse(401, "Unauthorized",
			"invalid payload signature", middleware.GetTraceID(c)))
		return
	}

	// ── Publish to Kafka ──────────────────────────────────────────────────────
	if err := h.producer.Publish(ctx, msg); err != nil {
		log.Error().Err(err).
			Str("ballot_type", req.BallotType).
			Int("province_id", int(req.ProvinceID)).
			Msg("ingestion: kafka publish failed")
		c.JSON(http.StatusServiceUnavailable, errorResponse(503, "Service Unavailable",
			"vote queue unavailable, please retry", middleware.GetTraceID(c)))
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status":          "accepted",
		"idempotency_key": req.IdempotencyKey,
	})
}

// parseEd25519PublicKey decodes a PKIX PEM-encoded Ed25519 public key.
func parseEd25519PublicKey(pemStr string) (ed25519.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse PKIX public key: %w", err)
	}
	ed, ok := pub.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("expected ed25519 public key, got %T", pub)
	}
	return ed, nil
}
