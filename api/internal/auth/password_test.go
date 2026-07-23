package auth

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"golang.org/x/crypto/argon2"
)

func TestHashAndVerifyPassword(t *testing.T) {
	t.Parallel()

	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$") {
		t.Fatalf("hash not in PHC argon2id format: %q", hash)
	}

	ok, err := VerifyPassword("correct horse battery staple", hash)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !ok {
		t.Fatal("correct password did not verify")
	}

	ok, err = VerifyPassword("wrong password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword(wrong): %v", err)
	}
	if ok {
		t.Fatal("wrong password verified")
	}
}

func TestHashPasswordSaltsAreUnique(t *testing.T) {
	t.Parallel()

	a, err := HashPassword("same input")
	if err != nil {
		t.Fatal(err)
	}
	b, err := HashPassword("same input")
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("two hashes of the same password are identical; salt is not random")
	}
}

func TestVerifyPasswordRejectsMalformedHashes(t *testing.T) {
	t.Parallel()

	cases := []string{
		"",
		"not a hash",
		"$argon2i$v=19$m=19456,t=2,p=1$c2FsdA$aGFzaA",  // wrong variant
		"$argon2id$v=18$m=19456,t=2,p=1$c2FsdA$aGFzaA", // wrong version
		"$argon2id$v=19$m=19456,t=2,p=1$!!!$aGFzaA",    // bad salt b64
	}
	for _, c := range cases {
		if _, err := VerifyPassword("whatever", c); err == nil {
			t.Errorf("VerifyPassword(%q): want error, got nil", c)
		}
	}
}

// TestVerifyPasswordUsesEmbeddedParams pins the property that lets us
// raise the package constants later without breaking stored hashes: a
// hash produced with different (weaker) parameters must still verify,
// because Verify reads params from the hash, not from the constants.
func TestVerifyPasswordUsesEmbeddedParams(t *testing.T) {
	t.Parallel()

	salt := []byte("0123456789abcdef")
	key := argon2.IDKey([]byte("password123"), salt, 1, 1024, 1, 32)
	weak := fmt.Sprintf("$argon2id$v=%d$m=1024,t=1,p=1$%s$%s",
		argon2.Version,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)

	ok, err := VerifyPassword("password123", weak)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !ok {
		t.Fatal("hash with non-default params did not verify; params not read from hash")
	}
}
