package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	// SessionTTL is how long a session lives without activity. Activity
	// slides the expiry forward (see Authenticate), so active users are
	// never logged out.
	SessionTTL = 7 * 24 * time.Hour

	// Sessions are extended when less than half the TTL remains, rather
	// than on every request, to keep authenticated reads write-free.
	sessionExtendThreshold = SessionTTL / 2

	minPasswordLen = 8
	// Bounded to keep Argon2 input sane; well past any real passphrase.
	maxPasswordLen = 512
	maxDisplayName = 100
	maxEmailLen    = 254
)

// ErrInvalidCredentials covers both unknown email and wrong password —
// the API never reveals which.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ValidationError marks user-fixable input problems; the handler returns
// these as 400s with the message intact.
type ValidationError struct{ msg string }

func (e ValidationError) Error() string { return e.msg }

func invalidf(format string, args ...any) error {
	return ValidationError{msg: fmt.Sprintf(format, args...)}
}

type Service struct {
	store  *Store
	logger *slog.Logger
}

func NewService(store *Store, logger *slog.Logger) *Service {
	return &Service{store: store, logger: logger}
}

// Register creates the user and an initial session in one step.
func (s *Service) Register(ctx context.Context, email, displayName, password string) (User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	displayName = strings.TrimSpace(displayName)

	if _, err := mail.ParseAddress(email); err != nil || len(email) > maxEmailLen {
		return User{}, "", invalidf("invalid email address")
	}
	if displayName == "" || utf8.RuneCountInString(displayName) > maxDisplayName {
		return User{}, "", invalidf("display name must be 1-%d characters", maxDisplayName)
	}
	if len(password) < minPasswordLen || len(password) > maxPasswordLen {
		return User{}, "", invalidf("password must be %d-%d characters", minPasswordLen, maxPasswordLen)
	}

	hash, err := HashPassword(password)
	if err != nil {
		return User{}, "", err
	}
	user, err := s.store.CreateUser(ctx, email, hash, displayName)
	if err != nil {
		return User{}, "", err // ErrEmailTaken or internal
	}

	token, err := s.startSession(ctx, user.ID)
	if err != nil {
		return User{}, "", err
	}
	s.logger.InfoContext(ctx, "user registered", "user_id", user.ID)
	return user, token, nil
}

// dummyHash is verified against when login hits an unknown email, so the
// request costs one Argon2 evaluation either way (timing-oracle hygiene).
var dummyHash = sync.OnceValue(func() string {
	h, err := HashPassword("dummy-timing-equalizer")
	if err != nil {
		panic(err) // crypto/rand failure; nothing sensible to do
	}
	return h
})

func (s *Service) Login(ctx context.Context, email, password string) (User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.store.UserByEmail(ctx, email)
	if errors.Is(err, ErrNotFound) {
		_, _ = VerifyPassword(password, dummyHash())
		return User{}, "", ErrInvalidCredentials
	}
	if err != nil {
		return User{}, "", err
	}

	ok, err := VerifyPassword(password, user.PasswordHash)
	if err != nil {
		// A hash we can't parse is corrupt data, not a bad password.
		return User{}, "", fmt.Errorf("verify password for user %s: %w", user.ID, err)
	}
	if !ok {
		return User{}, "", ErrInvalidCredentials
	}

	token, err := s.startSession(ctx, user.ID)
	if err != nil {
		return User{}, "", err
	}
	user.PasswordHash = ""
	s.logger.InfoContext(ctx, "user logged in", "user_id", user.ID)
	return user, token, nil
}

func (s *Service) startSession(ctx context.Context, userID string) (string, error) {
	raw, hash, err := NewSessionToken()
	if err != nil {
		return "", err
	}
	if err := s.store.CreateSession(ctx, hash, userID, time.Now().Add(SessionTTL)); err != nil {
		return "", err
	}
	return raw, nil
}

// Authenticate resolves a raw session token to its user, sliding the
// expiry forward when the session is past the midpoint of its TTL.
// The second return reports whether the expiry was extended (so the
// caller can refresh the cookie's Max-Age to match).
func (s *Service) Authenticate(ctx context.Context, rawToken string) (User, bool, error) {
	hash := HashToken(rawToken)
	user, expiresAt, err := s.store.SessionUser(ctx, hash)
	if err != nil {
		return User{}, false, err // ErrNotFound or internal
	}

	extended := false
	if time.Until(expiresAt) < sessionExtendThreshold {
		if err := s.store.ExtendSession(ctx, hash, time.Now().Add(SessionTTL)); err != nil {
			// The user is still authenticated; losing one extension is
			// harmless. Log and continue.
			s.logger.WarnContext(ctx, "extend session failed", "error", err)
		} else {
			extended = true
		}
	}
	return user, extended, nil
}

// Logout revokes the session server-side (instant revocation is the point
// of session storage over JWTs).
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	return s.store.DeleteSession(ctx, HashToken(rawToken))
}
