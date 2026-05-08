// Package auth provides password hashing utilities using the argon2id algorithm
// (RFC 9106 winner-takes-all recommendation for new password hashing).
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Parameters used for argon2id hashing.
// Dev values: fast enough for test/demo, still resistant to brute-force.
// Production should use m=65536, t=3, p=4 at minimum.
const (
	argonMemory      = 64 * 1024 // 64 MB
	argonIterations  = 3
	argonParallelism = 2
	argonSaltLen     = 16
	argonKeyLen      = 32
)

// ErrInvalidHash is returned when the stored hash format is unrecognised.
var ErrInvalidHash = errors.New("argon2: invalid hash format")

// HashPassword derives an argon2id hash of password and returns the encoded
// string (includes algorithm, version, params, salt, and hash — all needed
// for future verification).
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("argon2: generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, argonKeyLen)

	// Format: argon2id$<version>$<memory>,<iterations>,<parallelism>$<salt_b64>$<hash_b64>
	encoded := fmt.Sprintf("argon2id$%d$%d,%d,%d$%s$%s",
		argon2.Version,
		argonMemory, argonIterations, argonParallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
	return encoded, nil
}

// VerifyPassword reports whether password matches the stored encoded hash.
// Uses constant-time comparison to prevent timing attacks.
func VerifyPassword(password, encoded string) (bool, error) {
	memory, iterations, parallelism, salt, hash, err := decodeHash(encoded)
	if err != nil {
		return false, err
	}

	derived := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, argonKeyLen)
	return subtle.ConstantTimeCompare(hash, derived) == 1, nil
}

func decodeHash(encoded string) (memory uint32, iterations uint32, parallelism uint8, salt, hash []byte, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 || parts[0] != "argon2id" {
		return 0, 0, 0, nil, nil, ErrInvalidHash
	}

	var version int
	if _, err = fmt.Sscanf(parts[1], "%d", &version); err != nil {
		return 0, 0, 0, nil, nil, fmt.Errorf("argon2: parse version: %w", err)
	}

	var mem, iters int
	var threads int
	if _, err = fmt.Sscanf(parts[2], "%d,%d,%d", &mem, &iters, &threads); err != nil {
		return 0, 0, 0, nil, nil, fmt.Errorf("argon2: parse params: %w", err)
	}
	memory = uint32(mem)
	iterations = uint32(iters)
	parallelism = uint8(threads)

	if salt, err = base64.RawStdEncoding.DecodeString(parts[3]); err != nil {
		return 0, 0, 0, nil, nil, fmt.Errorf("argon2: decode salt: %w", err)
	}
	if hash, err = base64.RawStdEncoding.DecodeString(parts[4]); err != nil {
		return 0, 0, 0, nil, nil, fmt.Errorf("argon2: decode hash: %w", err)
	}
	return memory, iterations, parallelism, salt, hash, nil
}
