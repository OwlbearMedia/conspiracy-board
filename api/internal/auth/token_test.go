package auth

import "testing"

func TestNewSessionToken(t *testing.T) {
	t.Parallel()

	raw, hash, err := NewSessionToken()
	if err != nil {
		t.Fatalf("NewSessionToken: %v", err)
	}
	if raw == "" || hash == "" {
		t.Fatal("empty token or hash")
	}
	if raw == hash {
		t.Fatal("raw token equals its hash")
	}
	if got := HashToken(raw); got != hash {
		t.Fatalf("HashToken(raw) = %q, want %q", got, hash)
	}
	if len(hash) != 64 { // hex sha-256
		t.Fatalf("hash length = %d, want 64", len(hash))
	}

	raw2, _, err := NewSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	if raw == raw2 {
		t.Fatal("two tokens are identical")
	}
}
