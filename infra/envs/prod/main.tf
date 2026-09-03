# Long-lived production environment. Skeleton — build out in step 6 of the
# build order (ARCHITECTURE.md §10). The outputs listed at the bottom form the
# contract that envs/preview consumes; keep them stable.

terraform {
  required_version = ">= 1.7"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.62"
    }
  }
  backend "s3" {
    # bucket = "conspiracy-board-tfstate"
    # key    = "prod.tfstate"
    # region = "us-east-1"
    # dynamodb_table = "conspiracy-board-tflock"
  }
}

provider "aws" {
  region = var.region
}

variable "region" {
  type    = string
  default = "us-east-1"
}

variable "domain" {
  description = "Apex domain, e.g. example.com. App at app.<domain>, API at api.<domain>, previews at *.preview.<domain>"
  type        = string
}

# TODO(step 6): VPC + subnets, ALB + HTTPS listener with ACM certs for
# api.<domain> and *.preview.<domain>, ECS cluster + api service, RDS
# (db.t4g.micro, private), S3 + CloudFront (OAC), ECR repo with lifecycle
# policy (expire preview-* tags after 14 days), CloudWatch alarms, and the
# GitHub OIDC provider + deploy role scoped to this repo.

# --- outputs consumed by envs/preview (the shared-infra contract) -----------
# output "vpc_id"                 { value = module.network.vpc_id }
# output "private_subnet_ids"     { value = module.network.private_subnet_ids }
# output "ecs_cluster_arn"        { value = aws_ecs_cluster.this.arn }
# output "alb_https_listener_arn" { value = aws_lb_listener.https.arn }
# output "alb_dns_name"           { value = aws_lb.this.dns_name }
# output "alb_zone_id"            { value = aws_lb.this.zone_id }
# output "hosted_zone_id"         { value = data.aws_route53_zone.this.zone_id }
# output "preview_domain"         { value = "preview.${var.domain}" }
# output "api_security_group_id"  { value = aws_security_group.api.id }
# output "ecs_execution_role_arn" { value = aws_iam_role.ecs_execution.arn }
# output "ecs_task_role_arn"      { value = aws_iam_role.ecs_task.arn }
# output "db_admin_secret_arn"    { value = aws_secretsmanager_secret.db_admin.arn }
# output "rds_address"            { value = aws_db_instance.this.address }
