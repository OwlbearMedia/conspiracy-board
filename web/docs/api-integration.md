# Talking to the API

This is the contract the frontend relies on. The full endpoint list and event protocol are in [architecture.md §6](../../docs/architecture.md#6-api--real-time-design). The server side is described in [api/docs/conventions.md](../../api/docs/conventions.md).

## Base URL

Call the API with **relative paths under `/api/v1`** (e.g. `fetch('/api/v1/me')`). Locally, nginx makes this same-origin. In preview environments the Go binary serves both the SPA and the API.

> **Open question for production:** the design puts the SPA on `app.<domain>` and the API on `api.<domain>`, which is cross-origin. Either CloudFront gets an `/api/*` behavior routing to the ALB (so relative paths keep working), or the web build needs an absolute API base URL plus CORS and cookie-domain changes. See [project-status.md](../../docs/project-status.md#open-design-questions). Keep all API calls going through one small client module so this stays a one-line change.

## Requests and errors

- Send JSON bodies with `Content-Type: application/json`. Bodies are capped at 1 MiB.
- Every error response has the same shape:

  ```ts
  type ApiError = { error: { code: string; message: string } }
  ```

  Branch on `code` (stable), and show `message` only as a fallback. Codes so far: `invalid_json`, `invalid_input`, `email_taken`, `invalid_credentials`, `unauthenticated`, `csrf_mismatch`, `internal`.
- A `401 unauthenticated` from any endpoint means the session is gone. Clear local user state and route to login.

## Auth: cookies and CSRF

Implemented on the `auth` branch, following [ADR-3](../../docs/architecture.md#adr-3-self-implemented-cookie-sessions-over-cognitoauth0--accepted).

- **Session:** `POST /api/v1/auth/register` or `/login` sets two cookies. The frontend never handles the session token directly.
  - `cb_session`: HttpOnly, so JS can't read it. The browser sends it automatically.
  - `cb_csrf`: readable by JS.
- **CSRF:** on every mutating request (POST/PATCH/PUT/DELETE, except register and login), read `cb_csrf` from `document.cookie` and send it back as the **`X-CSRF-Token`** header. Otherwise the API responds `403 csrf_mismatch`.
- **Current user:** `GET /api/v1/me` returns `{ id, email, display_name, created_at }` or a 401. Call it on app start to restore the session.
- **Logout:** `POST /api/v1/auth/logout` (with the CSRF header) returns 204 and clears both cookies.
- Sessions slide: active users stay logged in, and they expire after 7 days of inactivity.

Suggested client shape: one `api()` wrapper that adds the JSON headers and the CSRF header on mutating methods, parses the error envelope into a typed error, and emits an "unauthenticated" signal on 401 for the auth store and router guard to handle.

## Real-time (planned, build step 4)

- Connect to `GET /api/v1/boards/{id}/ws` (relative, using `ws:` or `wss:` to match the page protocol). The session cookie authenticates it.
- Load the snapshot first with `GET /api/v1/boards/{id}`, which includes `seq`. Apply WS events idempotently by `seq`. On reconnect, send `last_seq`. If the server says so, re-fetch the snapshot.
- Mutations are optimistic in the Pinia `BoardStore` and roll back on server rejection. Pin drags send ephemeral moves and commit a single `pin.moved` on drop.
