package session

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

const tokenBytes = 32

var ErrInvalidToken = errors.New("invalid session token")

// Generate creates a cryptographically-random opaque bearer token and the
// SHA-256 digest that is safe to persist server-side. The raw bearer token
// must only be returned to the authenticated client and must never be logged
// or stored in PostgreSQL.
func Generate() (raw string, digest [sha256.Size]byte, err error) {
	buf := make([]byte, tokenBytes)
	if _, err = rand.Read(buf); err != nil {
		return "", digest, err
	}

	raw = base64.RawURLEncoding.EncodeToString(buf)
	digest = sha256.Sum256([]byte(raw))
	return raw, digest, nil
}

// Hash validates the encoded bearer token shape and returns the digest used
// for constant-format database lookup. Authentication code still performs
// all session expiry/revocation/user checks server-side.
func Hash(raw string) ([sha256.Size]byte, error) {
	var zero [sha256.Size]byte
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil || len(decoded) != tokenBytes {
		return zero, ErrInvalidToken
	}
	return sha256.Sum256([]byte(raw)), nil
}
