package auth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/dylanwhitney/conspiracy-board/api/internal/httpx"
)

const (
	// SessionCookie carries the opaque session token.
	// HttpOnly — script must never read it.
	SessionCookie = "cb_session"
	// CSRFCookie carries the double-submit token. Deliberately NOT
	// HttpOnly: the frontend reads it and echoes it back in CSRFHeader
	// on mutating requests. An attacker's cross-origin page can make the
	// browser *send* our cookies but cannot *read* them, so it cannot
	// forge the header.
	CSRFCookie = "cb_csrf"
	CSRFHeader = "X-CSRF-Token"
)

// Handler exposes the auth endpoints. secureCookies is false only in
// local dev (no TLS on localhost).
type Handler struct {
	svc           *Service
	logger        *slog.Logger
	secureCookies bool
}

func NewHandler(svc *Service, logger *slog.Logger, secureCookies bool) *Handler {
	return &Handler{svc: svc, logger: logger, secureCookies: secureCookies}
}

// Routes returns the router mounted at /api/v1/auth.
//
// register and login are deliberately outside the CSRF check: the caller
// has no token yet, and neither endpoint acts on an existing session.
// Login CSRF is mitigated by SameSite=Lax. logout mutates authenticated
// state, so it is protected.
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Post("/register", h.register)
	r.Post("/login", h.login)
	r.With(RequireCSRF).Post("/logout", h.logout)
	return r
}

type credentialsIn struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name,omitempty"`
	Password    string `json:"password"`
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var in credentialsIn
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", "request body must be a JSON object")
		return
	}

	user, token, err := h.svc.Register(r.Context(), in.Email, in.DisplayName, in.Password)
	if err != nil {
		var ve ValidationError
		switch {
		case errors.As(err, &ve):
			httpx.Error(w, http.StatusBadRequest, "invalid_input", ve.Error())
		case errors.Is(err, ErrEmailTaken):
			httpx.Error(w, http.StatusConflict, "email_taken", "that email is already registered")
		default:
			h.internalError(w, r, "register", err)
		}
		return
	}

	h.issueCookies(w, r, token)
	httpx.JSON(w, http.StatusCreated, user)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var in credentialsIn
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", "request body must be a JSON object")
		return
	}

	user, token, err := h.svc.Login(r.Context(), in.Email, in.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			// One message for unknown email and wrong password alike.
			httpx.Error(w, http.StatusUnauthorized, "invalid_credentials", "email or password is incorrect")
		} else {
			h.internalError(w, r, "login", err)
		}
		return
	}

	h.issueCookies(w, r, token)
	httpx.JSON(w, http.StatusOK, user)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(SessionCookie); err == nil && c.Value != "" {
		if err := h.svc.Logout(r.Context(), c.Value); err != nil {
			h.internalError(w, r, "logout", err)
			return
		}
	}
	h.clearCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

// Me returns the authenticated user; mount behind RequireUser.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFrom(r.Context())
	if !ok {
		// RequireUser guarantees a user; reaching this is a wiring bug.
		httpx.Error(w, http.StatusInternalServerError, "internal", "no user in context")
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

// issueCookies sets the session and CSRF cookies. Both are rotated on
// every login/register (session fixation hygiene).
func (h *Handler) issueCookies(w http.ResponseWriter, r *http.Request, sessionToken string) {
	csrf, err := newCSRFToken()
	if err != nil {
		// Extraordinarily unlikely (crypto/rand). Session still works;
		// the first mutating request will fail CSRF and the client can
		// re-login. Log loudly.
		h.logger.ErrorContext(r.Context(), "csrf token generation failed", "error", err)
	}
	http.SetCookie(w, h.sessionCookie(sessionToken, int(SessionTTL.Seconds())))
	http.SetCookie(w, h.csrfCookie(csrf, int(SessionTTL.Seconds())))
}

func (h *Handler) clearCookies(w http.ResponseWriter) {
	http.SetCookie(w, h.sessionCookie("", -1))
	http.SetCookie(w, h.csrfCookie("", -1))
}

func (h *Handler) sessionCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookie,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
	}
}

func (h *Handler) csrfCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     CSRFCookie,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: false, // read by the frontend, see CSRFCookie doc
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
	}
}

func (h *Handler) internalError(w http.ResponseWriter, r *http.Request, op string, err error) {
	h.logger.ErrorContext(r.Context(), op+" failed", "error", err)
	httpx.Error(w, http.StatusInternalServerError, "internal", "something went wrong")
}

// RefreshSessionCookie re-issues the session cookie with a full Max-Age.
// Called by RequireUser when the server-side expiry slides, so the cookie
// lifetime tracks the session lifetime.
func (h *Handler) refreshSessionCookie(w http.ResponseWriter, raw string) {
	http.SetCookie(w, h.sessionCookie(raw, int(SessionTTL.Seconds())))
}
