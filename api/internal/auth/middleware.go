package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"

	"github.com/dylanwhitney/conspiracy-board/api/internal/httpx"
)

type contextKey struct{ name string }

var userContextKey = contextKey{"auth.user"}

// UserFrom returns the authenticated user placed in the context by
// RequireUser.
func UserFrom(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(userContextKey).(User)
	return u, ok
}

// RequireUser authenticates the session cookie, injects the user into the
// request context, and refreshes the cookie when the session expiry
// slides. Unauthenticated requests get a 401 with a single generic code —
// no distinction between missing, unknown, and expired sessions.
func (h *Handler) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(SessionCookie)
		if err != nil || c.Value == "" {
			unauthorized(w)
			return
		}

		user, extended, err := h.svc.Authenticate(r.Context(), c.Value)
		if errors.Is(err, ErrNotFound) {
			unauthorized(w)
			return
		}
		if err != nil {
			h.internalError(w, r, "authenticate", err)
			return
		}
		if extended {
			h.refreshSessionCookie(w, c.Value)
		}

		next.ServeHTTP(w, r.WithContext(
			context.WithValue(r.Context(), userContextKey, user),
		))
	})
}

// RequireCSRF enforces the double-submit check on mutating methods: the
// CSRFHeader value must equal the CSRFCookie value. Safe methods pass
// through, as does any future CORS preflight.
func RequireCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}

		c, err := r.Cookie(CSRFCookie)
		header := r.Header.Get(CSRFHeader)
		if err != nil || c.Value == "" || header == "" ||
			subtle.ConstantTimeCompare([]byte(c.Value), []byte(header)) != 1 {
			httpx.Error(w, http.StatusForbidden, "csrf_mismatch",
				"missing or invalid CSRF token; send the "+CSRFCookie+" cookie value in the "+CSRFHeader+" header")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func unauthorized(w http.ResponseWriter) {
	httpx.Error(w, http.StatusUnauthorized, "unauthenticated", "authentication required")
}
