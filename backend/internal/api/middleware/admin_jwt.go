package middleware

import (
	"crypto/rsa"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/th-election/backend/internal/domain/voting"
)

// Context keys for admin JWT claims.
const (
	ContextKeyAdminID       = "admin_id"
	ContextKeyAdminUsername = "admin_username"
	ContextKeyAdminRole     = "admin_role"
)

// AdminJWT validates an admin RS256 JWT from the Authorization: Bearer header
// and enforces that the role claim equals "admin".
//
// IP whitelist is intentionally omitted for the portfolio — in production this
// would additionally gate access to a known CIDR range (e.g. VPN/office IPs).
func AdminJWT(publicKey *rsa.PublicKey) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := voting.ParseAdminJWT(publicKey, tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		if claims.Role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin role required"})
			return
		}

		c.Set(ContextKeyAdminID, claims.Subject)
		c.Set(ContextKeyAdminUsername, claims.Username)
		c.Set(ContextKeyAdminRole, claims.Role)
		c.Next()
	}
}
