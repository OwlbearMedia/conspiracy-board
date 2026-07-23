package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Sentinel errors surfaced by the store; the service maps these to API
// error codes.
var (
	ErrNotFound   = errors.New("not found")
	ErrEmailTaken = errors.New("email already registered")
)

// User is a row in users. PasswordHash is only populated by lookups that
// need it (login); it is never serialized.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	DisplayName  string    `json:"display_name"`
	CreatedAt    time.Time `json:"created_at"`
	PasswordHash string    `json:"-"`
}

// Store gives the auth service its database access. IDs cross the boundary
// as strings (uuid::text) to keep the dependency set minimal — no uuid
// library until something needs one.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const pgUniqueViolation = "23505"

func (s *Store) CreateUser(ctx context.Context, email, passwordHash, displayName string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, display_name)
		VALUES ($1, $2, $3)
		RETURNING id::text, email::text, display_name, created_at`,
		email, passwordHash, displayName,
	).Scan(&u.ID, &u.Email, &u.DisplayName, &u.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return User{}, ErrEmailTaken
		}
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

// UserByEmail returns the user including the password hash, for login.
func (s *Store) UserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, email::text, display_name, created_at, password_hash
		FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.DisplayName, &u.CreatedAt, &u.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("user by email: %w", err)
	}
	return u, nil
}

func (s *Store) CreateSession(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (token_hash, user_id, expires_at)
		VALUES ($1, $2, $3)`,
		tokenHash, userID, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// SessionUser resolves a live (unexpired) session to its user and the
// session's current expiry. Expired sessions are indistinguishable from
// absent ones.
func (s *Store) SessionUser(ctx context.Context, tokenHash string) (User, time.Time, error) {
	var (
		u         User
		expiresAt time.Time
	)
	err := s.pool.QueryRow(ctx, `
		SELECT u.id::text, u.email::text, u.display_name, u.created_at, s.expires_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > now()`,
		tokenHash,
	).Scan(&u.ID, &u.Email, &u.DisplayName, &u.CreatedAt, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, time.Time{}, ErrNotFound
	}
	if err != nil {
		return User{}, time.Time{}, fmt.Errorf("session user: %w", err)
	}
	return u, expiresAt, nil
}

// ExtendSession implements the sliding-expiry write. It is a no-op on
// expired or missing sessions.
func (s *Store) ExtendSession(ctx context.Context, tokenHash string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE sessions SET expires_at = $2
		WHERE token_hash = $1 AND expires_at > now()`,
		tokenHash, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("extend session: %w", err)
	}
	return nil
}

// DeleteSession is idempotent; deleting an unknown token is not an error.
func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// DeleteExpiredSessions is housekeeping; rows past expiry are already
// unusable (SessionUser filters on expiry), this just reclaims space.
// Intended to be called periodically once a janitor exists.
func (s *Store) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at <= now()`)
	if err != nil {
		return 0, fmt.Errorf("delete expired sessions: %w", err)
	}
	return tag.RowsAffected(), nil
}
