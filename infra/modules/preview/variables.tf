variable "slug" { type = string }
variable "image" { type = string }

variable "vpc_id" { type = string }
variable "private_subnet_ids" { type = list(string) }
variable "cluster_arn" { type = string }
variable "listener_arn" { type = string }
variable "alb_dns_name" { type = string }
variable "alb_zone_id" { type = string }
variable "hosted_zone_id" { type = string }
variable "preview_domain" { type = string } # e.g. preview.example.com
variable "api_sg_id" { type = string }
variable "execution_role_arn" { type = string }
variable "task_role_arn" { type = string }
variable "db_admin_secret" { type = string }
variable "db_host" { type = string }

variable "listener_rule_priority" {
  type        = number
  default     = null # let AWS pick; collisions across previews are avoided by hashing in main.tf
  description = "Optional explicit ALB rule priority"
}
