package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests by method, path, and status code.",
	}, []string{"method", "path", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_ms",
		Help:    "HTTP request duration in milliseconds (P50/P95/P99).",
		Buckets: []float64{5, 10, 25, 50, 100, 250, 500, 1000, 2500},
	}, []string{"method", "path"})

	voteIngestionTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "vote_ingestion_total",
		Help: "Total votes ingested, by source and ballot_type.",
	}, []string{"source", "ballot_type", "status"})

	onlineVoteCastTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "online_vote_cast_total",
		Help: "Total online votes cast, by ballot_type and status.",
	}, []string{"ballot_type", "status"})

	wsConnectionsActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ws_connections_active",
		Help: "Active WebSocket connections.",
	})
)

// Metrics is Gin middleware that records RED (Rate/Errors/Duration) metrics per route.
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath() // use route pattern, not raw path (avoids cardinality explosion)
		if path == "" {
			path = "unknown"
		}

		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path).
			Observe(float64(time.Since(start).Milliseconds()))
	}
}

// RecordVoteIngestion records a vote ingestion event (called from ingestion handler).
func RecordVoteIngestion(source, ballotType, status string) {
	voteIngestionTotal.WithLabelValues(source, ballotType, status).Inc()
}

// RecordOnlineVoteCast records an online vote cast event.
func RecordOnlineVoteCast(ballotType, status string) {
	onlineVoteCastTotal.WithLabelValues(ballotType, status).Inc()
}

// SetWSConnections sets the active WebSocket gauge.
func SetWSConnections(n float64) {
	wsConnectionsActive.Set(n)
}
