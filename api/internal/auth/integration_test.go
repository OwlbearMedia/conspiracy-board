// Integration tests: the full HTTP stack (router, middleware, handlers,
// store) against a real Postgres. Skipped unless DATABASE_URL is set; CI
// provides a postgres service container, locally `make up` provides one.
package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dylanwhitney/conspiracy-board/api/internal/auth"
	"github.com/dylanwhitney/conspiracy-board/api/internal/config"
	"github.com/dylanwhitney/conspiracy-board/api/internal/migrations"
	"github.com/dylanwhitney/conspiracy-board/api/internal/server"
)

func newTestServer(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	if err := migrations.Up(dbURL); err != nil {
		t.Fatalf("migrations: %v", err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	// Each test run starts clean; sessions/boards cascade from users.
	if _, err := pool.Exec(ctx, `TRUNCATE users CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := config.Config{Port: 0, DatabaseURL: dbURL, Env: "local"}

	ts := httptest.NewServer(server.New(logger, pool, cfg))
	t.Cleanup(ts.Close)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar: %v", err)
	}
	return ts, &http.Client{Jar: jar}
}

func postJSON(t *testing.T, client *http.Client, url string, body map[string]any, csrf string) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if csrf != "" {
		req.Header.Set(auth.CSRFHeader, csrf)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func decodeBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return m
}

func csrfFromJar(t *testing.T, client *http.Client, serverURL string) string {
	t.Helper()
	u, err := url.Parse(serverURL)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range client.Jar.Cookies(u) {
		if c.Name == auth.CSRFCookie {
			return c.Value
		}
	}
	return ""
}

func TestAuthFlow(t *testing.T) {
	ts, client := newTestServer(t)
	api := ts.URL + "/api/v1"

	register := map[string]any{
		"email":        "alice@example.com",
		"display_name": "Alice",
		"password":     "hunter2hunter2",
	}

	// Register: 201, user JSON, session + csrf cookies set.
	resp := postJSON(t, client, api+"/auth/register", register, "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d, want 201", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	if body["email"] != "alice@example.com" || body["display_name"] != "Alice" {
		t.Fatalf("register body = %v", body)
	}
	if _, hasHash := body["password_hash"]; hasHash {
		t.Fatal("password_hash leaked in register response")
	}
	if csrfFromJar(t, client, ts.URL) == "" {
		t.Fatal("no CSRF cookie after register")
	}

	// /me with the fresh session: 200.
	resp, err := client.Get(api + "/me")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/me status = %d, want 200", resp.StatusCode)
	}
	if body := decodeBody(t, resp); body["email"] != "alice@example.com" {
		t.Fatalf("/me body = %v", body)
	}

	// Duplicate registration: 409.
	resp = postJSON(t, client, api+"/auth/register", register, "")
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate register status = %d, want 409", resp.StatusCode)
	}
	resp.Body.Close()

	// Same email, different case: still 409 (citext).
	upper := map[string]any{
		"email":        "ALICE@example.com",
		"display_name": "Alice2",
		"password":     "hunter2hunter2",
	}
	resp = postJSON(t, client, api+"/auth/register", upper, "")
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("case-variant register status = %d, want 409", resp.StatusCode)
	}
	resp.Body.Close()

	// Logout without CSRF header: 403, session untouched.
	resp = postJSON(t, client, api+"/auth/logout", nil, "")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("logout without csrf status = %d, want 403", resp.StatusCode)
	}
	resp.Body.Close()

	// Logout with CSRF header: 204.
	resp = postJSON(t, client, api+"/auth/logout", nil, csrfFromJar(t, client, ts.URL))
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("logout status = %d, want 204", resp.StatusCode)
	}
	resp.Body.Close()

	// Session is revoked server-side: /me now 401.
	resp, err = client.Get(api + "/me")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("/me after logout status = %d, want 401", resp.StatusCode)
	}
	resp.Body.Close()

	// Login with wrong password: 401.
	resp = postJSON(t, client, api+"/auth/login", map[string]any{
		"email": "alice@example.com", "password": "wrong-password",
	}, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad login status = %d, want 401", resp.StatusCode)
	}
	resp.Body.Close()

	// Login with unknown email: same 401, same error code.
	resp = postJSON(t, client, api+"/auth/login", map[string]any{
		"email": "nobody@example.com", "password": "wrong-password",
	}, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unknown email login status = %d, want 401", resp.StatusCode)
	}
	resp.Body.Close()

	// Correct login: 200, session works again.
	resp = postJSON(t, client, api+"/auth/login", map[string]any{
		"email": "Alice@Example.com", "password": "hunter2hunter2",
	}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	resp, err = client.Get(api + "/me")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/me after login status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestRegisterValidation(t *testing.T) {
	ts, client := newTestServer(t)
	api := ts.URL + "/api/v1"

	cases := []struct {
		name string
		body map[string]any
	}{
		{"missing email", map[string]any{"display_name": "A", "password": "hunter2hunter2"}},
		{"bad email", map[string]any{"email": "not-an-email", "display_name": "A", "password": "hunter2hunter2"}},
		{"short password", map[string]any{"email": "a@example.com", "display_name": "A", "password": "short"}},
		{"empty display name", map[string]any{"email": "a@example.com", "display_name": "  ", "password": "hunter2hunter2"}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			resp := postJSON(t, client, api+"/auth/register", tt.body, "")
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", resp.StatusCode)
			}
		})
	}

	t.Run("malformed json", func(t *testing.T) {
		resp, err := client.Post(api+"/auth/register", "application/json", bytes.NewBufferString("{nope"))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", resp.StatusCode)
		}
	})
}

func TestMeRequiresAuth(t *testing.T) {
	ts, client := newTestServer(t)

	resp, err := client.Get(ts.URL + "/api/v1/me")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("/me unauthenticated status = %d, want 401", resp.StatusCode)
	}
}
