package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashAndVerify(t *testing.T) {
	password := "S3cr3t!Pass"

	hash, err := HashPassword(password)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(hash, "argon2id$"), "hash should have argon2id prefix")

	ok, err := VerifyPassword(password, hash)
	require.NoError(t, err)
	assert.True(t, ok, "correct password should verify")
}

func TestVerifyWrongPassword(t *testing.T) {
	hash, err := HashPassword("correct-password")
	require.NoError(t, err)

	ok, err := VerifyPassword("wrong-password", hash)
	require.NoError(t, err)
	assert.False(t, ok, "wrong password must not verify")
}

func TestVerifyInvalidHash(t *testing.T) {
	_, err := VerifyPassword("password", "not-a-valid-hash")
	assert.ErrorIs(t, err, ErrInvalidHash)
}

func TestHashesAreDifferent(t *testing.T) {
	// Same input → different hashes (unique random salt each time).
	h1, err := HashPassword("same")
	require.NoError(t, err)
	h2, err := HashPassword("same")
	require.NoError(t, err)
	assert.NotEqual(t, h1, h2, "hashes must differ due to unique salt")
}
