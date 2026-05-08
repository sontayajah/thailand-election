package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const TraceIDKey = "trace_id"
const TraceIDHeader = "X-Trace-ID"

// TraceID injects a unique request trace ID into every request context and response header.
func TraceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(TraceIDHeader)
		if traceID == "" {
			traceID = uuid.New().String()
		}
		c.Set(TraceIDKey, traceID)
		c.Header(TraceIDHeader, traceID)
		c.Next()
	}
}

// GetTraceID retrieves the trace ID from the Gin context.
func GetTraceID(c *gin.Context) string {
	if v, ok := c.Get(TraceIDKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
