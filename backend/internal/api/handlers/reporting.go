package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/th-election/backend/internal/api/middleware"
	"github.com/th-election/backend/internal/cache"
	"github.com/th-election/backend/internal/config"
	"github.com/th-election/backend/internal/domain/reporting"
	db "github.com/th-election/backend/internal/db/sqlc"
)

// ReportingHandler serves all public read-only election reporting endpoints (B-02 – B-09).
type ReportingHandler struct {
	cfg      *config.Config
	queries  db.Querier
	redis    *cache.Clients
	cbClient *cache.CircuitBreakerClient
}

func NewReportingHandler(
	cfg *config.Config,
	queries db.Querier,
	redis *cache.Clients,
	cbClient *cache.CircuitBreakerClient,
) *ReportingHandler {
	return &ReportingHandler{cfg: cfg, queries: queries, redis: redis, cbClient: cbClient}
}

// ── B-07: GET /api/v1/provinces ───────────────────────────────────────────

func (h *ReportingHandler) ListProvinces(c *gin.Context) {
	ctx := c.Request.Context()

	// Try Redis cache first
	if cached, err := h.redis.Cache.Get(ctx, cache.KeyCacheProvincesList()).Result(); err == nil && cached != "" {
		c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(cached))
		return
	}

	provinces, err := h.queries.ListProvinces(ctx)
	if err != nil {
		h.internalError(c, "list provinces", err)
		return
	}

	type provinceDTO struct {
		ID                int16  `json:"id"`
		NameTH            string `json:"name_th"`
		NameEN            string `json:"name_en"`
		Region            string `json:"region"`
		ConstituencyCount int16  `json:"constituency_count"`
		EligibleVoters    int32  `json:"eligible_voters"`
		SvgPathID         string `json:"svg_path_id,omitempty"`
	}

	dtos := make([]provinceDTO, len(provinces))
	for i, p := range provinces {
		svgID := ""
		if p.SvgPathID != nil {
			svgID = *p.SvgPathID
		}
		dtos[i] = provinceDTO{
			ID:                p.ID,
			NameTH:            p.NameTh,
			NameEN:            p.NameEn,
			Region:            p.Region,
			ConstituencyCount: p.ConstituencyCount,
			EligibleVoters:    p.EligibleVoters,
			SvgPathID:         svgID,
		}
	}

	payload, _ := json.Marshal(gin.H{"provinces": dtos, "total": len(dtos)})
	_ = h.redis.Cache.Set(ctx, cache.KeyCacheProvincesList(), string(payload), cache.KeyTTLProvinces).Err()
	c.Data(http.StatusOK, "application/json; charset=utf-8", payload)
}

// ── B-08: GET /api/v1/parties ─────────────────────────────────────────────

func (h *ReportingHandler) ListParties(c *gin.Context) {
	ctx := c.Request.Context()

	parties, err := h.queries.ListParties(ctx)
	if err != nil {
		h.internalError(c, "list parties", err)
		return
	}

	type partyDTO struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		ShortName string `json:"short_name"`
		ColorHex  string `json:"color_hex"`
		LogoURL   string `json:"logo_url,omitempty"`
	}
	dtos := make([]partyDTO, len(parties))
	for i, p := range parties {
		logo := ""
		if p.LogoUrl != nil {
			logo = *p.LogoUrl
		}
		dtos[i] = partyDTO{
			ID:        p.ID.String(),
			Name:      p.Name,
			ShortName: p.ShortName,
			ColorHex:  p.ColorHex,
			LogoURL:   logo,
		}
	}
	c.JSON(http.StatusOK, gin.H{"parties": dtos})
}

// ── B-08: GET /api/v1/candidates ─────────────────────────────────────────

func (h *ReportingHandler) ListCandidates(c *gin.Context) {
	ctx := c.Request.Context()

	if cached, err := h.redis.Cache.Get(ctx, cache.KeyCacheCandidatesList()).Result(); err == nil && cached != "" {
		c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(cached))
		return
	}

	candidates, err := h.queries.ListPartyListCandidates(ctx)
	if err != nil {
		h.internalError(c, "list candidates", err)
		return
	}

	payload, _ := json.Marshal(gin.H{"candidates": candidates, "total": len(candidates)})
	_ = h.redis.Cache.Set(ctx, cache.KeyCacheCandidatesList(), string(payload), cache.KeyTTLCandidates).Err()
	c.Data(http.StatusOK, "application/json; charset=utf-8", payload)
}

// ── B-02: GET /api/v1/election/national/summary ───────────────────────────

func (h *ReportingHandler) NationalSummary(c *gin.Context) {
	ctx := c.Request.Context()

	parties, err := h.getPartyListNational(ctx)
	if err != nil {
		h.internalError(c, "get national leaderboard", err)
		return
	}

	referendum, err := h.getReferendumNational(ctx)
	if err != nil {
		h.internalError(c, "get national referendum", err)
		return
	}

	var totalVotes int64
	for _, p := range parties {
		totalVotes += p.PartyListVotes
	}

	c.JSON(http.StatusOK, reporting.NationalSummaryResponse{
		Parties:         parties,
		TotalVotesCast:  totalVotes,
		ReferendumBreak: referendum,
		UpdatedAt:       time.Now().UTC().Format(time.RFC3339),
	})
}

// ── B-03: GET /api/v1/election/provinces/:id/summary ─────────────────────

func (h *ReportingHandler) ProvinceSummary(c *gin.Context) {
	provinceID, err := strconv.ParseInt(c.Param("id"), 10, 16)
	if err != nil {
		h.badRequest(c, "province id must be a number")
		return
	}
	ctx := c.Request.Context()
	pid := int16(provinceID)

	ballotType := db.BallotType(c.DefaultQuery("ballot_type", "CONSTITUENCY"))

	province, err := h.queries.GetProvinceByID(ctx, pid)
	if err != nil {
		h.notFound(c, "province not found")
		return
	}

	summaries, err := h.queries.GetProvinceSummary(ctx, db.GetProvinceSummaryParams{
		ProvinceID: pid,
		BallotType: ballotType,
	})
	if err != nil {
		h.internalError(c, "get province summary", err)
		return
	}

	entries := make([]reporting.ProvinceResultEntry, len(summaries))
	for i, s := range summaries {
		entry := reporting.ProvinceResultEntry{
			TotalVotes: s.TotalVotes,
		}
			if s.PartyID.Valid {
			entry.PartyID = fmt.Sprintf("%x-%x-%x-%x-%x",
				s.PartyID.Bytes[0:4], s.PartyID.Bytes[4:6],
				s.PartyID.Bytes[6:8], s.PartyID.Bytes[8:10],
				s.PartyID.Bytes[10:16])
		}
		if s.PartyName != nil {
			entry.PartyName = *s.PartyName
		}
		if s.PartyShortName != nil {
			entry.PartyShortName = *s.PartyShortName
		}
		if s.PartyColor != nil {
			entry.PartyColor = *s.PartyColor
		}
		if s.FullName != nil {
			entry.CandidateName = *s.FullName
		}
		entries[i] = entry
	}

	c.JSON(http.StatusOK, reporting.ProvinceSummaryResponse{
		ProvinceID:   pid,
		ProvinceName: province.NameEn,
		BallotType:   string(ballotType),
		Results:      entries,
	})
}

// ── B-04: GET /api/v1/election/provinces/:id/constituencies ──────────────

func (h *ReportingHandler) ProvinceConstituencies(c *gin.Context) {
	provinceID, err := strconv.ParseInt(c.Param("id"), 10, 16)
	if err != nil {
		h.badRequest(c, "province id must be a number")
		return
	}
	ctx := c.Request.Context()
	pid := int16(provinceID)

	constituencies, err := h.queries.ListConstituenciesByProvince(ctx, pid)
	if err != nil {
		h.internalError(c, "list constituencies", err)
		return
	}

	type constituencyResult struct {
		ID             string                        `json:"id"`
		Name           string                        `json:"name"`
		EligibleVoters int32                         `json:"eligible_voters"`
		ConstituencyNo int16                         `json:"constituency_no"`
		TopCandidates  []db.GetConstituencySummaryRow `json:"top_candidates"`
	}

	results := make([]constituencyResult, len(constituencies))
	for i, con := range constituencies {
		summary, _ := h.queries.GetConstituencySummary(ctx, con.ID)
		if len(summary) > 5 {
			summary = summary[:5]
		}
		results[i] = constituencyResult{
			ID:             con.ID.String(),
			Name:           con.Name,
			EligibleVoters: con.EligibleVoters,
			ConstituencyNo: con.ConstituencyNo,
			TopCandidates:  summary,
		}
	}

	c.JSON(http.StatusOK, gin.H{"province_id": pid, "constituencies": results})
}

// ── B-05: GET /api/v1/election/party-list/calculate ──────────────────────

func (h *ReportingHandler) PartyListCalculation(c *gin.Context) {
	ctx := c.Request.Context()

	if cached, err := h.redis.Cache.Get(ctx, cache.KeyCachePartySeats()).Result(); err == nil && cached != "" {
		c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(cached))
		return
	}

	rows, err := h.queries.GetPartyListNational(ctx)
	if err != nil {
		h.internalError(c, "get party list", err)
		return
	}

	input := make([]reporting.PartyVotes, len(rows))
	var totalVotes int64
	for i, r := range rows {
		input[i] = reporting.PartyVotes{
			PartyID:        r.PartyID.String(),
			PartyName:      r.PartyName,
			PartyShortName: r.PartyShortName,
			PartyColor:     r.PartyColor,
			TotalVotes:     r.TotalVotes,
		}
		totalVotes += r.TotalVotes
	}

	allocs, votesPerSeat := reporting.CalculatePartyListSeats(input)

	resp := reporting.PartyListCalculationResponse{
		TotalPartyListVotes: totalVotes,
		VotesPerSeat:        votesPerSeat,
		Allocations:         allocs,
		UpdatedAt:           time.Now().UTC().Format(time.RFC3339),
	}

	payload, _ := json.Marshal(resp)
	_ = h.redis.Cache.Set(ctx, cache.KeyCachePartySeats(), string(payload), cache.KeyTTLPartySeats).Err()
	c.Data(http.StatusOK, "application/json; charset=utf-8", payload)
}

// ── B-06: GET /api/v1/election/referendum/summary ────────────────────────

func (h *ReportingHandler) ReferendumSummary(c *gin.Context) {
	ctx := c.Request.Context()

	national, err := h.getReferendumNational(ctx)
	if err != nil {
		h.internalError(c, "get referendum national", err)
		return
	}

	provinces, err := h.queries.ListProvinces(ctx)
	if err != nil {
		h.internalError(c, "list provinces for referendum", err)
		return
	}

	byProvince := make([]reporting.ProvinceReferendumResult, 0, len(provinces))
	for _, prov := range provinces {
		row, err := h.queries.GetReferendumProvinceSummary(ctx, prov.ID)
		if err != nil {
			continue
		}
		pct := 0.0
		if row.TotalVotes > 0 {
			pct = float64(row.AgreeVotes) / float64(row.TotalVotes) * 100
		}
		byProvince = append(byProvince, reporting.ProvinceReferendumResult{
			ProvinceID:    prov.ID,
			ProvinceName:  prov.NameEn,
			AgreeVotes:    row.AgreeVotes,
			DisagreeVotes: row.DisagreeVotes,
			AbstainVotes:  row.AbstainVotes,
			TotalVotes:    row.TotalVotes,
			AgreePct:      pct,
		})
	}

	c.JSON(http.StatusOK, reporting.ReferendumSummaryResponse{
		National:   national,
		ByProvince: byProvince,
	})
}

// ── Private helpers ───────────────────────────────────────────────────────

func (h *ReportingHandler) getPartyListNational(ctx context.Context) ([]reporting.PartyNationalResult, error) {
	rows, err := h.queries.GetPartyListNational(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]reporting.PartyNationalResult, len(rows))
	for i, r := range rows {
		results[i] = reporting.PartyNationalResult{
			PartyID:        r.PartyID.String(),
			PartyName:      r.PartyName,
			PartyShortName: r.PartyShortName,
			PartyColor:     r.PartyColor,
			PartyListSeats: int(r.SeatCount),
			PartyListVotes: r.TotalVotes,
			TotalSeats:     int(r.SeatCount),
		}
	}
	return results, nil
}

func (h *ReportingHandler) getReferendumNational(ctx context.Context) (reporting.ReferendumBreakdown, error) {
	// Try CB-protected Redis first
	vals, err := h.cbClient.HGetAll(ctx, cache.KeyNationalReferendum)
	if err == nil && len(vals) > 0 {
		agree, _ := strconv.ParseInt(vals["agree"], 10, 64)
		disagree, _ := strconv.ParseInt(vals["disagree"], 10, 64)
		abstain, _ := strconv.ParseInt(vals["abstain"], 10, 64)
		total := agree + disagree + abstain
		pct := 0.0
		if total > 0 {
			pct = float64(agree) / float64(total) * 100
		}
		return reporting.ReferendumBreakdown{
			AgreeVotes: agree, DisagreeVotes: disagree, AbstainVotes: abstain,
			TotalVotes: total, AgreePct: pct,
		}, nil
	}

	// Fallback to PostgreSQL (circuit breaker open or cache miss)
	row, err := h.queries.GetReferendumNationalSummary(ctx)
	if err != nil {
		return reporting.ReferendumBreakdown{}, err
	}
	agreePct := row.AgreePct
	disagreePct := row.DisagreePct
	return reporting.ReferendumBreakdown{
		AgreeVotes:    row.AgreeVotes,
		DisagreeVotes: row.DisagreeVotes,
		AbstainVotes:  row.AbstainVotes,
		TotalVotes:    row.TotalVotes,
		AgreePct:      agreePct,
		DisagreePct:   disagreePct,
	}, nil
}

func (h *ReportingHandler) internalError(c *gin.Context, op string, err error) {
	traceID := middleware.GetTraceID(c)
	log.Error().Err(err).Str("traceId", traceID).Str("op", op).Msg("handler error")
	c.JSON(http.StatusInternalServerError, errorResponse(500, "Internal Server Error", op+" failed", traceID))
}

func (h *ReportingHandler) badRequest(c *gin.Context, detail string) {
	c.JSON(http.StatusBadRequest, errorResponse(400, "Bad Request", detail, middleware.GetTraceID(c)))
}

func (h *ReportingHandler) notFound(c *gin.Context, detail string) {
	c.JSON(http.StatusNotFound, errorResponse(404, "Not Found", detail, middleware.GetTraceID(c)))
}

func errorResponse(status int, title, detail, traceID string) gin.H {
	return gin.H{
		"type":    "https://errors.thailand-election.example.com/error",
		"title":   title,
		"status":  status,
		"detail":  detail,
		"traceId": traceID,
	}
}
