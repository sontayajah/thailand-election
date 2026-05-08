package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// Recovery catches panics and returns a 500 JSON response with the trace ID.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				traceID := GetTraceID(c)
				log.Error().
					Str("traceId", traceID).
					Interface("panic", err).
					Str("path", c.Request.URL.Path).
					Msg("panic_recovered")

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"type":    "https://errors.thailand-election.example.com/internal-error",
					"title":   "Internal Server Error",
					"status":  500,
					"detail":  "An unexpected error occurred. Please try again later.",
					"traceId": traceID,
				})
			}
		}()
		c.Next()
	}
}
