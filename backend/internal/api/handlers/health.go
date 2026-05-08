package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/th-election/backend/internal/cache"
	"github.com/th-election/backend/internal/config"
)

// HealthHandler implements GET /api/v1/health (B-09).
type HealthHandler struct {
	cfg   *config.Config
	pool  *pgxpool.Pool
	redis *cache.Clients
}

func NewHealthHandler(cfg *config.Config, pool *pgxpool.Pool, redis *cache.Clients) *HealthHandler {
	return &HealthHandler{cfg: cfg, pool: pool, redis: redis}
}

type healthStatus struct {
	Status    string                 `json:"status"`
	Timestamp string                 `json:"timestamp"`
	Services  map[string]serviceInfo `json:"services"`
}

type serviceInfo struct {
	Status  string `json:"status"`
	Latency string `json:"latency,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Health checks all downstream dependencies and returns a summary.
// Returns 200 if everything is healthy, 503 if any critical service is down.
func (h *HealthHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	services := make(map[string]serviceInfo)
	allHealthy := true

	services["postgres"] = checkPing(ctx, func() error {
		return h.pool.Ping(ctx)
	})
	services["redis-persistent"] = checkRedisPing(ctx, h.redis.Persistent)
	services["redis-cache"] = checkRedisPing(ctx, h.redis.Cache)
	services["redis-asynq"] = checkRedisPing(ctx, h.redis.Asynq)

	for _, svc := range services {
		if svc.Status != "ok" {
			allHealthy = false
			break
		}
	}

	status := "healthy"
	httpStatus := http.StatusOK
	if !allHealthy {
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, healthStatus{
		Status:    status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Services:  services,
	})
}

func checkPing(ctx context.Context, fn func() error) serviceInfo {
	start := time.Now()
	if err := fn(); err != nil {
		return serviceInfo{Status: "error", Error: err.Error()}
	}
	return serviceInfo{Status: "ok", Latency: time.Since(start).String()}
}

func checkRedisPing(ctx context.Context, client *redis.Client) serviceInfo {
	start := time.Now()
	if err := client.Ping(ctx).Err(); err != nil {
		return serviceInfo{Status: "error", Error: err.Error()}
	}
	return serviceInfo{Status: "ok", Latency: time.Since(start).String()}
}
