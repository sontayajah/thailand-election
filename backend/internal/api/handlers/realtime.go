package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/th-election/backend/internal/config"
)

// RealtimeHandler issues Centrifugo connection tokens (B-10).
type RealtimeHandler struct {
	cfg *config.Config
}

func NewRealtimeHandler(cfg *config.Config) *RealtimeHandler {
	return &RealtimeHandler{cfg: cfg}
}

// IssueToken creates a short-lived HMAC JWT for anonymous Centrifugo connections.
// Public viewers do not need voter authentication — any visitor can subscribe.
// POST /api/v1/realtime/token
func (h *RealtimeHandler) IssueToken(c *gin.Context) {
	claims := jwt.MapClaims{
		"sub": "public-viewer",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(h.cfg.Centrifugo.TokenSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token":      signed,
		"expires_in": 3600,
	})
}
