# Web

Vue 3 + TypeScript single-page app for Conspiracy Board, built with Vite and using Pinia for state. The board canvas will use Konva via `vue-konva` ([architecture.md §6, Frontend structure](../docs/architecture.md#frontend-structure)).

**Current state:** scaffold only. `App.vue` shows whether the API is ready. There's no router, stores, API client, or canvas yet. See [docs/project-status.md](../docs/project-status.md).

## Run it

The recommended way is the full stack from the repo root: `make up`, then open http://localhost:8080. Vite runs in a container behind nginx, so `/api/*` calls work same-origin and HMR works through the proxy.

To run on the host (Node 20+):

```sh
corepack enable   # installs the pnpm version pinned in package.json ("packageManager")
pnpm install
pnpm dev          # http://localhost:5173
```

> Vite on `:5173` has **no `/api` proxy**, so API calls fail from that origin. Use the Compose stack for anything that talks to the API, or add a `server.proxy` for `/api` (with `ws: true`) in `vite.config.ts`.

## Scripts

| Script | What it does |
|---|---|
| `pnpm dev` | Vite dev server on `:5173` (`host: true` so nginx can reach it) |
| `pnpm typecheck` | `vue-tsc -b`, the same check CI runs |
| `pnpm build` | Type check, then production build into `dist/` |
| `pnpm preview` | Serve the built `dist/` locally |

From the repo root, `make typecheck` runs the type check.

## Layout

```
index.html        Vite entry HTML
src/main.ts       creates the app, installs Pinia
src/App.vue       root component (currently the API status page)
src/env.d.ts      Vite + *.vue type shims
vite.config.ts    dev server config (port 5173, strictPort)
tsconfig.json     strict TS, bundler resolution
```

As features land, the planned structure (per the design) is:

```
src/
  api/            typed fetch wrapper + endpoint functions (see docs/api-integration.md)
  stores/         Pinia stores: auth/session, boards list, BoardStore (snapshot + WS events)
  views/          route-level pages: Login, Register, Dashboard, Board
  components/     shared UI; board/ for Konva canvas pieces
  router.ts       vue-router routes + auth guard
```

Update this section when the real structure exists.

## How it ships

- **Production:** `pnpm build` → `dist/` synced to S3 and served via CloudFront ([ADR-2](../docs/architecture.md#adr-2-drop-nginx-from-production-s3--cloudfront-for-the-frontend--accepted)).
- **Preview environments:** the `api/Dockerfile` `web-build` stage builds this app and embeds it in the Go binary, which serves it with `SERVE_STATIC=1`.

Either way the build must not depend on anything outside `web/`, and CI runs `pnpm install --frozen-lockfile`, so always commit `pnpm-lock.yaml` together with any `package.json` change.

## More docs

- [docs/api-integration.md](docs/api-integration.md): API paths, error envelope, cookies and CSRF, WebSocket
