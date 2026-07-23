# Infrastructure

Terraform for all AWS resources. Layout:

```
infra/
├── envs/
│   ├── prod/      # long-lived: VPC, ALB, ECS cluster+service, RDS, S3/CloudFront, ECR, OIDC role
│   └── preview/   # ephemeral: one workspace per preview branch (see ADR-6)
└── modules/
    └── preview/   # ECS service + ALB host rule + DNS record + per-preview database
```

## State

Remote state in S3 with DynamoDB locking (create the bucket/table once by hand
or with a tiny bootstrap config). `envs/preview` relies on Terraform
**workspaces** — one per active preview — so each preview's state is isolated
and `terraform destroy` on a workspace removes exactly that preview.

## Shared vs ephemeral

Previews deliberately reuse the expensive resources from `prod`:

| Shared (prod-owned)                     | Per-preview (module) |
|----------------------------------------|----------------------|
| VPC, subnets, security groups          | ECS service + task definition (desired 1) |
| ALB + HTTPS listener                   | ALB listener rule (host header match) |
| Wildcard cert `*.preview.<domain>` (ACM) | Route 53 record `<slug>.preview.<domain>` |
| RDS instance                           | Database `preview_<slug>` on that instance |
| ECS cluster, ECR repo                  | |

`envs/preview` reads prod outputs via `terraform_remote_state`.

## Order of operations

1. Bootstrap state bucket + lock table
2. `envs/prod`: `terraform init && terraform apply` (fill in `terraform.tfvars`)
3. Set repo variables `AWS_DEPLOY_ROLE_ARN` and `APP_DOMAIN` in GitHub
4. Push a `preview/*` branch — the workflow does the rest
