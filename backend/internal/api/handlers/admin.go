package handlers

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
	"github.com/th-election/backend/internal/api/middleware"
	"github.com/th-election/backend/internal/config"
	db "github.com/th-election/backend/internal/db/sqlc"
	"github.com/th-election/backend/internal/domain/auth"
	"github.com/th-election/backend/internal/domain/voting"
	vkafka "github.com/th-election/backend/internal/kafka"
)

// AdminHandler handles admin-only endpoints: login, batch vote ingestion, audit logs.
type AdminHandler struct {
	cfg        *config.Config
	queries    db.Querier
	producer   *vkafka.Producer
	privateKey *rsa.PrivateKey
}

// NewAdminHandler wires up the admin handler.
func NewAdminHandler(
	cfg *config.Config,
	queries db.Querier,
	producer *vkafka.Producer,
	privateKey *rsa.PrivateKey,
) *AdminHandler {
	return &AdminHandler{
		cfg:        cfg,
		queries:    queries,
		producer:   producer,
		privateKey: privateKey,
	}
}

// ── POST /api/v1/admin/auth/login ─────────────────────────────────────────────

type adminLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login authenticates an admin user with username+password (argon2id), issues
// an RS256 JWT with role=admin, and writes an audit log entry.
func (h *AdminHandler) Login(c *gin.Context) {
	var req adminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	admin, err := h.queries.GetAdminByUsername(c.Request.Context(), req.Username)
	if err != nil {
		// Deliberately vague — don't confirm whether the username exists.
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	ok, err := auth.VerifyPassword(req.Password, admin.PasswordHash)
	if err != nil || !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Update last login timestamp (best-effort).
	_ = h.queries.UpdateAdminLastLogin(c.Request.Context(), admin.ID)

	// Audit log.
	_ = writeAuditLog(c, h.queries, admin.ID, "admin.login", nil, nil, nil)

	token, expiresAt, err := voting.SignAdminJWT(
		h.privateKey,
		admin.ID,
		admin.Username,
		admin.Role,
		h.cfg.JWT.AdminTTL,
	)
	if err != nil {
		log.Error().Err(err).Msg("admin login: sign jwt")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"expires_at": expiresAt.UTC().Format("2006-01-02T15:04:05Z"),
		"role":       admin.Role,
	})
}

// ── POST /api/v1/admin/votes/batch  (B-11) ───────────────────────────────────

// batchVoteEntry is a single vote record inside a batch request.
// Admin batch votes don't require an Ed25519 signature (auth is done by JWT).
type batchVoteEntry struct {
	BallotType     string `json:"ballot_type" binding:"required,oneof=CONSTITUENCY PARTY_LIST REFERENDUM"`
	ProvinceID     int16  `json:"province_id" binding:"required"`
	ConstituencyID string `json:"constituency_id"`
	CandidateID    string `json:"candidate_id"`
	ReferendumVote string `json:"referendum_vote"`
	VoteCount      int32  `json:"vote_count" binding:"required,min=1,max=10000"`
}

type batchVotesRequest struct {
	Votes []batchVoteEntry `json:"votes" binding:"required,min=1,max=1000,dive"`
}

// BatchVotes (B-11) publishes a batch of votes to Kafka as source=admin_batch.
// Intended for demo seeding and testing — never for production use.
func (h *AdminHandler) BatchVotes(c *gin.Context) {
	var req batchVotesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminIDStr := c.GetString(middleware.ContextKeyAdminID)
	adminID, _ := uuid.Parse(adminIDStr)

	var enqueued, failed int
	for i, v := range req.Votes {
		// Validate ballot-type-specific fields.
		if err := validateBatchEntry(v); err != nil {
			log.Warn().Err(err).Int("index", i).Msg("batch votes: skip invalid entry")
			failed++
			continue
		}

		msg := &vkafka.VoteMessage{
			BallotType:     v.BallotType,
			Source:         "admin_batch",
			ProvinceID:     v.ProvinceID,
			ConstituencyID: v.ConstituencyID,
			CandidateID:    v.CandidateID,
			ReferendumVote: v.ReferendumVote,
			VoteCount:      v.VoteCount,
			AnonymousToken: uuid.New().String(), // synthetic anonymous token per entry
			IdempotencyKey: fmt.Sprintf("admin_batch:%s:%d", adminIDStr, i),
		}

		if err := h.producer.Publish(c.Request.Context(), msg); err != nil {
			log.Error().Err(err).Int("index", i).Msg("batch votes: publish failed")
			failed++
			continue
		}
		enqueued++
	}

	// Audit log with batch metadata.
	meta, _ := json.Marshal(map[string]int{"enqueued": enqueued, "failed": failed, "total": len(req.Votes)})
	_ = writeAuditLog(c, h.queries, adminID, "admin.batch_votes", strPtr("vote"), nil, meta)

	status := http.StatusAccepted
	if enqueued == 0 {
		status = http.StatusUnprocessableEntity
	}
	c.JSON(status, gin.H{
		"enqueued": enqueued,
		"failed":   failed,
		"total":    len(req.Votes),
	})
}

func validateBatchEntry(v batchVoteEntry) error {
	switch v.BallotType {
	case "CONSTITUENCY":
		if v.ConstituencyID == "" || v.CandidateID == "" {
			return fmt.Errorf("constituency ballot requires constituency_id and candidate_id")
		}
	case "PARTY_LIST":
		if v.CandidateID == "" {
			return fmt.Errorf("party_list ballot requires candidate_id (party UUID)")
		}
	case "REFERENDUM":
		switch v.ReferendumVote {
		case "AGREE", "DISAGREE", "ABSTAIN":
			// valid
		default:
			return fmt.Errorf("referendum ballot requires referendum_vote AGREE|DISAGREE|ABSTAIN")
		}
	}
	return nil
}

// ── GET /api/v1/admin/audit-logs ─────────────────────────────────────────────

// ListAuditLogs returns a paginated list of audit log entries.
// Query params: limit (default 50, max 200), offset (default 0).
func (h *AdminHandler) ListAuditLogs(c *gin.Context) {
	limit := int32(50)
	offset := int32(0)

	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			limit = int32(n)
		}
	}
	if o := c.Query("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = int32(n)
		}
	}

	rows, err := h.queries.ListAuditLogs(c.Request.Context(), db.ListAuditLogsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		log.Error().Err(err).Msg("list audit logs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch audit logs"})
		return
	}

	type auditLogRow struct {
		ID            int64   `json:"id"`
		AdminUsername *string `json:"admin_username"`
		Action        string  `json:"action"`
		TargetType    *string `json:"target_type"`
		TargetID      *string `json:"target_id"`
		IPAddress     string  `json:"ip_address"`
		CreatedAt     string  `json:"created_at"`
	}

	out := make([]auditLogRow, len(rows))
	for i, r := range rows {
		out[i] = auditLogRow{
			ID:            r.ID,
			AdminUsername: r.AdminUsername,
			Action:        r.Action,
			TargetType:    r.TargetType,
			TargetID:      r.TargetID,
			IPAddress:     r.IpAddress.String(),
			CreatedAt:     r.CreatedAt.Time.UTC().Format("2006-01-02T15:04:05Z"),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":   out,
		"limit":  limit,
		"offset": offset,
	})
}

// ── Utility helpers ───────────────────────────────────────────────────────────

// writeAuditLog inserts an audit_logs record for the current request.
// Errors are logged but not propagated — audit logging must never break the main flow.
func writeAuditLog(
	c *gin.Context,
	q db.Querier,
	adminID uuid.UUID,
	action string,
	targetType *string,
	targetID *string,
	metadata []byte,
) error {
	ip, err := netip.ParseAddr(c.ClientIP())
	if err != nil {
		ip = netip.MustParseAddr("0.0.0.0")
	}
	ua := c.GetHeader("User-Agent")
	var uaPtr *string
	if ua != "" {
		uaPtr = &ua
	}
	if metadata == nil {
		metadata = []byte("{}")
	}

	if err := q.InsertAuditLog(c.Request.Context(), db.InsertAuditLogParams{
		AdminID:    pgtype.UUID{Bytes: adminID, Valid: true},
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		IpAddress:  ip,
		UserAgent:  uaPtr,
		Metadata:   metadata,
	}); err != nil {
		log.Warn().Err(err).Str("action", action).Msg("audit log: insert failed")
		return err
	}
	return nil
}

func strPtr(s string) *string { return &s }
