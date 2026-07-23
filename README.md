# Conspiracy Board

Collaborative cork boards: pin clues, connect them with strings, investigate together in real time.

**Stack:** Go (chi/pgx) · Vue 3 + TypeScript · PostgreSQL 16 · WebSockets · ECS Fargate · Terraform · GitHub Actions (OIDC)

See [ARCHITECTURE.md](ARCHITECTURE.md) for the full design — ADRs, data model, real-time protocol, and AWS topology.

## Local development

```sh
cp .env.example .env   # optional, defaults work
make up                # builds and starts postgres, api (hot reload), vite, nginx
```

App runs at http://localhost:8080 (nginx fronts both the API and the Vite dev server, same-origin like production). `make logs`, `make test`, `make typecheck`, `make down`.

## Repository layout

```
api/     Go API — chi router, pgx, embedded SQL migrations (`api migrate`)
web/     Vue 3 + TS + Pinia frontend (Vite)
nginx/   local-dev reverse proxy only (prod uses ALB + CloudFront)
infra/   Terraform — envs/prod (long-lived), envs/preview + modules/preview (ephemeral)
.github/ CI, production deploy, preview deploy/teardown workflows
```

## Preview environments

Push a branch named `preview/<name>` → a live environment appears at
`https://<name>.preview.<domain>` (single container: API + embedded SPA, own
database on the shared RDS). Delete the branch to tear it down; a nightly
sweep catches anything orphaned. Details in ARCHITECTURE.md, ADR-6.
