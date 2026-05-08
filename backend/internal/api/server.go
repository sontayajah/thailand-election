package api

import (
	"crypto/rsa"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/th-election/backend/internal/api/handlers"
	"github.com/th-election/backend/internal/api/middleware"
	"github.com/th-election/backend/internal/cache"
	"github.com/th-election/backend/internal/config"
	db "github.com/th-election/backend/internal/db/sqlc"
	vkafka "github.com/th-election/backend/internal/kafka"
)

// NewRouter builds and returns the fully configured Gin engine.
func NewRouter(
	cfg *config.Config,
	pool *pgxpool.Pool,
	queries db.Querier,
	redis *cache.Clients,
	cbClient *cache.CircuitBreakerClient,
	producer *vkafka.Producer,
	ovHandler *handlers.OnlineVotingHandler,
	adminHandler *handlers.AdminHandler,
	voterPublicKey *rsa.PublicKey,
) *gin.Engine {
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// ── Global middleware ────────────────────────────────────────────────────
	r.Use(middleware.Recovery())
	r.Use(middleware.TraceID())
	r.Use(middleware.Logger())
	r.Use(middleware.Metrics())
	r.Use(middleware.CORS([]string{
		"http://localhost:3000",
		"http://localhost:80",
		"http://localhost",
	}))

	// ── Prometheus metrics (internal — not behind Kong) ──────────────────────
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// ── Handler dependencies ─────────────────────────────────────────────────
	reportingH := handlers.NewReportingHandler(cfg, queries, redis, cbClient)
	healthH := handlers.NewHealthHandler(cfg, pool, redis)
	realtimeH := handlers.NewRealtimeHandler(cfg)
	ingestionH := handlers.NewIngestionHandler(cfg, queries, redis, producer)

	// ── API v1 ───────────────────────────────────────────────────────────────
	v1 := r.Group("/api/v1")
	{
		// Health check (B-09)
		v1.GET("/health", healthH.Health)

		// Public master data
		v1.GET("/provinces", reportingH.ListProvinces)
		v1.GET("/candidates", reportingH.ListCandidates)
		v1.GET("/parties", reportingH.ListParties)

		// Reporting endpoints (B-02 through B-08)
		election := v1.Group("/election")
		{
			election.GET("/national/summary", reportingH.NationalSummary)
			election.GET("/provinces/:id/summary", reportingH.ProvinceSummary)
			election.GET("/provinces/:id/constituencies", reportingH.ProvinceConstituencies)
			election.GET("/party-list/calculate", reportingH.PartyListCalculation)
			election.GET("/referendum/summary", reportingH.ReferendumSummary)
		}

		// Physical vote ingestion from polling stations (B-01)
		// Protected by Kong API-Key plugin in production.
		v1.POST("/votes", ingestionH.IngestVote)

		// Real-time WebSocket token (B-10)
		v1.POST("/realtime/token", realtimeH.IssueToken)

		// ── Admin API (B-11, audit logs) ─────────────────────────────────────
		admin := v1.Group("/admin")
		{
			// Login — returns admin RS256 JWT
			admin.POST("/auth/login", adminHandler.Login)

			// All subsequent admin routes require a valid admin JWT
			adminProtected := admin.Group("")
			adminProtected.Use(middleware.AdminJWT(voterPublicKey))
			{
				adminProtected.POST("/votes/batch", adminHandler.BatchVotes)  // B-11
				adminProtected.GET("/audit-logs", adminHandler.ListAuditLogs)
			}
		}

		// ── Online voting (B-12 through B-18) ───────────────────────────────
		ov := v1.Group("/online-voting")
		{
			// Unauthenticated auth flow
			ov.POST("/auth/verify-id", ovHandler.VerifyID)   // B-12
			ov.POST("/auth/request-otp", ovHandler.RequestOTP) // B-13
			ov.POST("/auth/verify-otp", ovHandler.VerifyOTP)   // B-14

			// Public receipt verification (no JWT required)
			ov.GET("/receipt/:hash", ovHandler.GetReceipt) // B-18

			// Voter-authenticated routes (RS256 JWT required)
			protected := ov.Group("")
			protected.Use(middleware.VoterJWT(voterPublicKey))
			{
				protected.GET("/eligibility", ovHandler.Eligibility)          // B-15
				protected.GET("/ballot/:ballot_type", ovHandler.GetBallot)    // B-16
				protected.POST("/cast", ovHandler.CastVote)                   // B-17
			}
		}
	}

	return r
}
