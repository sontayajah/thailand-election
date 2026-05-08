package voting

import (
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AdminClaims are the JWT claims for admin sessions.
// Contains a role claim so the middleware can gate endpoints by role.
type AdminClaims struct {
	jwt.RegisteredClaims
	Role     string `json:"role"`
	Username string `json:"username"`
}

// SignAdminJWT issues a signed RS256 JWT for an admin user.
func SignAdminJWT(
	privateKey *rsa.PrivateKey,
	adminID uuid.UUID,
	username string,
	role string,
	ttl time.Duration,
) (string, time.Time, error) {
	expiresAt := time.Now().Add(ttl)

	claims := AdminClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   adminID.String(),
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		Role:     role,
		Username: username,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(privateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign admin jwt: %w", err)
	}
	return signed, expiresAt, nil
}

// ParseAdminJWT validates an admin JWT and returns the embedded claims.
func ParseAdminJWT(publicKey *rsa.PublicKey, tokenStr string) (*AdminClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&AdminClaims{},
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return publicKey, nil
		},
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("parse admin jwt: %w", err)
	}

	claims, ok := token.Claims.(*AdminClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid admin claims")
	}
	return claims, nil
}
