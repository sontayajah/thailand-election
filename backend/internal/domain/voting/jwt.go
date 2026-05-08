package voting

import (
	"crypto/rsa"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const issuer = "thailand-election"

// VoterClaims are the JWT claims embedded in the anonymous voter token.
//
// SECURITY: NO national_id, phone, or any PII is stored here.
// The anonymous_token (avt) is a fresh UUID generated at OTP verification time.
// It links vote_events to each other (for receipt purposes) but NOT to voter identity.
type VoterClaims struct {
	jwt.RegisteredClaims

	// AnonymousVoteToken — fresh UUID per authenticated session.
	// Stored in vote_events.anonymous_token and vote_receipts.anonymous_token.
	// Never stored in voter_sessions or voter_registry.
	AnonymousToken string `json:"avt"`

	// Ballots the voter is eligible to cast. Always ["CONSTITUENCY","PARTY_LIST","REFERENDUM"].
	Ballots []string `json:"ballots"`

	// ProvinceID and ConstituencyID from voter_registry — used by the ballot endpoint.
	ProvinceID     int16  `json:"province_id"`
	ConstituencyID string `json:"constituency_id"`
}

// LoadPrivateKey reads an RSA private key from a PEM file.
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key %q: %w", path, err)
	}
	key, err := jwt.ParseRSAPrivateKeyFromPEM(b)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	return key, nil
}

// LoadPublicKey reads an RSA public key from a PEM file.
func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read public key %q: %w", path, err)
	}
	key, err := jwt.ParseRSAPublicKeyFromPEM(b)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	return key, nil
}

// SignVoterJWT creates a signed RS256 JWT for a voter session.
// The resulting token contains NO personally-identifiable information.
func SignVoterJWT(
	privateKey *rsa.PrivateKey,
	sessionID uuid.UUID,
	anonymousToken uuid.UUID,
	provinceID int16,
	constituencyID uuid.UUID,
	ttl time.Duration,
) (string, time.Time, error) {
	expiresAt := time.Now().Add(ttl)

	claims := VoterClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   sessionID.String(),
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		AnonymousToken: anonymousToken.String(),
		Ballots:        []string{"CONSTITUENCY", "PARTY_LIST", "REFERENDUM"},
		ProvinceID:     provinceID,
		ConstituencyID: constituencyID.String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(privateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign jwt: %w", err)
	}
	return signed, expiresAt, nil
}

// ParseVoterJWT validates a voter JWT and returns the embedded claims.
func ParseVoterJWT(publicKey *rsa.PublicKey, tokenStr string) (*VoterClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&VoterClaims{},
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
		return nil, fmt.Errorf("parse jwt: %w", err)
	}

	claims, ok := token.Claims.(*VoterClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid claims")
	}
	return claims, nil
}
