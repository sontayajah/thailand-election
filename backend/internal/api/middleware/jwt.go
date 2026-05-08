package middleware

import (
	"crypto/rsa"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/th-election/backend/internal/domain/voting"
)

// Context keys for voter JWT claims.
const (
	ContextKeySessionID      = "voter_session_id"
	ContextKeyAnonymousToken = "voter_anonymous_token"
	ContextKeyProvinceID     = "voter_province_id"
	ContextKeyConstituencyID = "voter_constituency_id"
)

// VoterJWT validates a voter RS256 JWT from the Authorization: Bearer <token> header.
// On success it stores the session_id, anonymous_token, province_id and constituency_id
// in the Gin context for downstream handlers.
func VoterJWT(publicKey *rsa.PublicKey) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := voting.ParseVoterJWT(publicKey, tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set(ContextKeySessionID, claims.Subject)
		c.Set(ContextKeyAnonymousToken, claims.AnonymousToken)
		c.Set(ContextKeyProvinceID, claims.ProvinceID)
		c.Set(ContextKeyConstituencyID, claims.ConstituencyID)
		c.Next()
	}
}
