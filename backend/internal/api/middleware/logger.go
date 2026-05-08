package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// Logger is a Gin middleware that emits a zerolog structured JSON record per request.
// Never logs: national_id, OTP, JWT payload, phone number, anonymous_vote_token.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		traceID := GetTraceID(c)

		event := log.Info()
		if status >= 500 {
			event = log.Error()
		} else if status >= 400 {
			event = log.Warn()
		}

		event.
			Str("traceId", traceID).
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", status).
			Int64("latency_ms", latency.Milliseconds()).
			Str("ip", c.ClientIP()).
			Int("bytes", c.Writer.Size()).
			Msg("http_request")
	}
}
