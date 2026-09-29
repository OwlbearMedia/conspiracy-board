# CI/CD

All automation is in [`.github/`](../.github/). The target design is in [architecture.md §8](architecture.md#8-cicd). This doc describes what exists now.

## Workflows

| Workflow | Trigger | What it does | State |
|---|---|---|---|
| [ci.yml](../.github/workflows/ci.yml) | every PR, push to `main` | **api**: `go vet`, `go build`, run migrations and `go test -race` against a Postgres service container. **web**: `pnpm install --frozen-lockfile`, `typecheck`, `build`. **docker**: build the prod image (no push) | Should pass once `go.sum` is committed; confirm on the next run |
| [deploy.yml](../.github/workflows/deploy.yml) | push to `main` | OIDC into AWS, build and push the image to ECR, then (placeholders) run migrations as a one-off ECS task, update the ECS service, and sync the SPA to S3 + invalidate CloudFront | Fails: no AWS yet (build step 6) |
| [preview-deploy.yml](../.github/workflows/preview-deploy.yml) | push to `preview/**` | Build an image with the SPA embedded, then `terraform apply` a per-slug workspace in `infra/envs/preview` | Needs prod infra |
| [preview-teardown.yml](../.github/workflows/preview-teardown.yml) | `preview/**` branch deleted; nightly 06:00 UTC; manual | Destroy that preview's workspace. The nightly sweep destroys any preview whose branch is gone | Nightly run fails: no AWS yet |
| [dependabot-automerge.yml](../.github/workflows/dependabot-automerge.yml) | every PR (acts only on Dependabot's) | Enable squash auto-merge for patch and minor bumps; majors wait for a human | Works, but see the branch protection warning below |

## Required repository settings

| Setting | Why |
|---|---|
| Variable `AWS_DEPLOY_ROLE_ARN` | IAM role that trusts GitHub OIDC, scoped to this repo (created by `infra/envs/prod`). No long-lived AWS keys anywhere |
| Variable `APP_DOMAIN` | Used to print preview URLs |
| Environment `production` | Used by `deploy.yml`. Add required reviewers here if you want a manual gate |
| Settings → General → **Allow auto-merge** | Needed by the Dependabot auto-merge workflow |
| Branch protection on `main` requiring `api`, `web`, `docker` | **Without this, auto-merge merges immediately whether CI is green or not.** That's what is happening today |

## Rules to preserve

- **`ci.yml` must stay secret-free.** Dependabot PRs run with a read-only token and no repository secrets. Any job that needs AWS credentials goes in a separate workflow guarded by `if: github.actor != 'dependabot[bot]'`. (Also noted in [dependabot.yml](../.github/dependabot.yml).)
- **Migrations run before rollout.** Deploy runs `api migrate` as a one-off task with the new image *before* updating the service, so a booting task always sees a compatible schema. Migrations must therefore be backward-compatible with the previous release (expand, then contract).
- **The same image serves traffic and runs migrations.** The prod image's entrypoint is `/bin/api`, and passing `migrate` as the argument switches modes.
- **Preview slug logic is duplicated** in `preview-deploy.yml` and `preview-teardown.yml` (twice there). Change all three together, and keep them consistent with the `slug` validation in `infra/envs/preview/main.tf`.

## Dependabot

[dependabot.yml](../.github/dependabot.yml) checks weekly for: Go modules (`/api`), npm/pnpm (`/web`), Docker base images (`api/Dockerfile` + Compose), GitHub Actions, and Terraform providers (both envs). Minor and patch updates are grouped per ecosystem; majors arrive as individual PRs and aren't auto-merged.

## Preview environments

Design: [ADR-6](architecture.md#adr-6-opt-in-ephemeral-preview-environments-per-branch--accepted). Terraform details: [infra/README.md](../infra/README.md).

- **Create:** push a branch named `preview/<name>`. It becomes `https://<slug>.preview.<APP_DOMAIN>`, where the slug is lowercased, non-alphanumerics become `-`, and it's capped at 30 characters (`preview/JIRA-42_fix` → `jira-42-fix`).
- **What you get:** one Fargate task running the prod image with `SERVE_STATIC=1` (API + embedded SPA), an ALB host rule, a Route 53 record, and database `preview_<slug>` on the shared RDS instance.
- **Destroy:** delete the branch, or run Preview Teardown manually with a `slug` input.
- **Cost:** about $9/month per preview left running. Don't create `preview/*` branches casually.
