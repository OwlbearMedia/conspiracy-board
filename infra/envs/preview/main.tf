terraform {
  required_version = ">= 1.7"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.61"
    }
  }
  backend "s3" {
    # bucket/key/region/dynamodb_table supplied via `terraform init -backend-config`
    # or hardcode once the bootstrap bucket exists, e.g.:
    # bucket         = "conspiracy-board-tfstate"
    # key            = "preview.tfstate"   # workspaces suffix this automatically
    # region         = "us-east-1"
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

variable "slug" {
  description = "DNS-safe preview identifier derived from the branch name"
  type        = string
  validation {
    condition     = can(regex("^[a-z0-9][a-z0-9-]{0,29}$", var.slug))
    error_message = "slug must be lowercase alphanumeric/dashes, max 30 chars"
  }
}

variable "image" {
  description = "Full ECR image URI to deploy"
  type        = string
}

data "terraform_remote_state" "prod" {
  backend = "s3"
  config = {
    # same backend bucket; prod writes its outputs here
    bucket = "conspiracy-board-tfstate"
    key    = "prod.tfstate"
    region = var.region
  }
}

module "preview" {
  source = "../../modules/preview"

  slug  = var.slug
  image = var.image

  # everything shared comes from prod outputs
  vpc_id             = data.terraform_remote_state.prod.outputs.vpc_id
  private_subnet_ids = data.terraform_remote_state.prod.outputs.private_subnet_ids
  cluster_arn        = data.terraform_remote_state.prod.outputs.ecs_cluster_arn
  listener_arn       = data.terraform_remote_state.prod.outputs.alb_https_listener_arn
  alb_dns_name       = data.terraform_remote_state.prod.outputs.alb_dns_name
  alb_zone_id        = data.terraform_remote_state.prod.outputs.alb_zone_id
  hosted_zone_id     = data.terraform_remote_state.prod.outputs.hosted_zone_id
  preview_domain     = data.terraform_remote_state.prod.outputs.preview_domain # preview.<domain>
  api_sg_id          = data.terraform_remote_state.prod.outputs.api_security_group_id
  execution_role_arn = data.terraform_remote_state.prod.outputs.ecs_execution_role_arn
  task_role_arn      = data.terraform_remote_state.prod.outputs.ecs_task_role_arn
  db_admin_secret    = data.terraform_remote_state.prod.outputs.db_admin_secret_arn
  db_host            = data.terraform_remote_state.prod.outputs.rds_address
}

output "url" {
  value = module.preview.url
}
