# Conspiracy Board

Collaborative cork boards: pin clues, connect them with strings, investigate together in real time.

**Stack:** Go (chi/pgx) · Vue 3 + TypeScript · PostgreSQL 16 · WebSockets · ECS Fargate · Terraform · GitHub Actions (OIDC)

> **Project status:** early implementation. The foundation (Compose stack, migrations, health checks, CI skeleton) is on `main`; auth is in progress on the `auth` branch. See [docs/project-status.md](docs/project-status.md) for what works, what doesn't, and what's next.

## Prerequisites

| Tool | Version | Needed for |
|---|---|---|
| [Docker](https://docs.docker.com/get-docker/) with Compose v2 | recent | the local stack (required) |
| `make` | any | shortcut commands (optional, but assumed below) |
| [Go](https://go.dev/dl/) | 1.26+ (see `api/go.mod`) | running tests / tooling outside Docker |
| [Node.js](https://nodejs.org/) + pnpm via `corepack enable` | Node 20+, pnpm pinned in `web/package.json` | running web tooling outside Docker |
| [Terraform](https://developer.hashicorp.com/terraform/install) | 1.7+ | infrastructure work only |

Only Docker is required to run the app. Go and Node are needed for running tests, type checks, and editor tooling on the host.

## Quick start

```sh
git clone <repo-url> conspiracy-board && cd conspiracy-board
cp .env.example .env   # optional; the defaults work
make up                # builds and starts postgres, migrate, api (hot reload), web (vite), nginx
```

Open http://localhost:8080. You should see **API status: ready**, which means the browser reached the API through nginx and the API reached Postgres.

## Everyday commands

| Command | What it does |
|---|---|
| `make up` | Build and start the full stack in the background |
| `make down` | Stop the stack (the database volume is kept) |
| `make logs` | Follow API logs |
| `make migrate` | Run database migrations against the local DB |
| `make test` | `go test -race ./...` in `api/` |
| `make lint` | `go vet ./...` in `api/` |
| `make typecheck` | `vue-tsc` type check in `web/` |

Local services:

| URL / port | Service |
|---|---|
| http://localhost:8080 | nginx: the app. `/api/*` goes to the Go API, everything else to Vite |
| `localhost:5432` | Postgres (user `conspiracy`, password from `.env` or `localdev`, db `conspiracy`) |

For running the API or web app outside Docker, resetting the database, and troubleshooting, see [docs/development.md](docs/development.md).

## Repository layout

```
api/     Go API: chi router, pgx, embedded SQL migrations    → api/README.md
web/     Vue 3 + TS + Pinia frontend (Vite)                   → web/README.md
docs/    Project-wide docs: architecture, status, dev, CI/CD → docs/README.md
infra/   Terraform: envs/prod (long-lived), envs/preview + modules/preview (ephemeral) → infra/README.md
nginx/   Local-dev reverse proxy only (prod uses ALB + CloudFront)
.github/ CI, production deploy, preview deploy/teardown, Dependabot
```

## Documentation

- [docs/](docs/README.md): index of all project documentation
- [docs/architecture.md](docs/architecture.md): design, ADRs, data model, real-time protocol, AWS topology
- [docs/project-status.md](docs/project-status.md): progress, known issues, next steps
- [docs/development.md](docs/development.md): local development in depth
- [docs/ci-cd.md](docs/ci-cd.md): workflows, Dependabot, preview environments
- [api/README.md](api/README.md) and [web/README.md](web/README.md): per-app guides
- [AGENTS.md](AGENTS.md): guidance for AI coding agents (also a handy summary for humans)

## Preview environments

Push a branch named `preview/<name>` and a live environment appears at
`https://<name>.preview.<domain>` (a single container serving API + embedded SPA, with its own
database on the shared RDS). Delete the branch to tear it down; a nightly
sweep catches anything orphaned. **This needs the AWS infrastructure from build step 6, which
doesn't exist yet.** Details: [docs/ci-cd.md](docs/ci-cd.md#preview-environments) and
[ADR-6](docs/architecture.md#adr-6-opt-in-ephemeral-preview-environments-per-branch--accepted).
