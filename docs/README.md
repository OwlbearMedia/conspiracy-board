# Documentation

Docs for both human engineers and AI agents. Project-wide docs live here, and docs that only matter inside one app live next to that app.

## Project-wide (`docs/`)

| Doc | Read it when |
|---|---|
| [architecture.md](architecture.md) | You need the design: goals, ADRs, data model, REST + WebSocket protocol, AWS topology, build order |
| [project-status.md](project-status.md) | You're picking the project back up: what's done, what's broken, what's next |
| [development.md](development.md) | You're setting up or debugging the local environment |
| [ci-cd.md](ci-cd.md) | You're touching GitHub Actions, Dependabot, deploys, or preview environments |

## App-specific

| Doc | Scope |
|---|---|
| [api/README.md](../api/README.md) | Running, configuring, and testing the Go API |
| [api/docs/conventions.md](../api/docs/conventions.md) | Go code layout, error handling, logging, testing patterns |
| [api/docs/database.md](../api/docs/database.md) | Schema, migrations, query style |
| [web/README.md](../web/README.md) | Running and building the Vue frontend |
| [web/docs/api-integration.md](../web/docs/api-integration.md) | How the frontend talks to the API (paths, errors, cookies/CSRF, WebSocket) |
| [infra/README.md](../infra/README.md) | Terraform layout, state, shared vs per-preview resources |

## Where new docs go

- **Affects more than one of `api/`, `web/`, `infra/`?** Put it in `docs/`.
- **Only matters inside one app?** Put it in `<app>/docs/` and link it from that app's README and the table above.
- **A significant design decision?** Add an ADR section to [architecture.md](architecture.md#3-key-decisions-adr-summaries) (numbered, with its status). If the ADR list gets long, split it into `docs/adr/NNNN-title.md`.
- **Did the change make a doc wrong?** Fix the doc in the same change. [project-status.md](project-status.md) especially goes stale quickly.
