package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// Session tokens are opaque: 32 bytes from crypto/rand, base64url-encoded
// for the cookie. Only the SHA-256 of the token is stored server-side
// (sessions.token_hash), so a leaked database dump cannot be replayed as
// live sessions. SHA-256 (not Argon2) is appropriate here because the input
// already has 256 bits of entropy — there is nothing to brute-force.

const tokenBytes = 32

// NewSessionToken returns the raw token to set in the cookie and the hash
// to store in the sessions table.
func NewSessionToken() (raw, hash string, err error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate session token: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, HashToken(raw), nil
}

// HashToken maps a raw token to its storage form.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// newCSRFToken returns a random double-submit CSRF token.
func newCSRFToken() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate csrf token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
