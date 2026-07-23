locals {
  name = "preview-${var.slug}"
  host = "${var.slug}.${var.preview_domain}"
  # Deterministic ALB rule priority in [1000, 45000) derived from the slug,
  # keeping previews clear of hand-assigned low priorities.
  priority = coalesce(
    var.listener_rule_priority,
    1000 + parseint(substr(sha1(var.slug), 0, 4), 16) % 44000
  )
  # Preview DB name: underscores, since Postgres identifiers dislike dashes.
  db_name = "preview_${replace(var.slug, "-", "_")}"
}

# --- database on the shared RDS instance -----------------------------------
# The migration container creates the schema; this null_resource creates the
# database itself. For a personal project this is pragmatic; a hardened setup
# would use a Lambda-backed custom resource instead of local-exec + psql.
resource "null_resource" "database" {
  triggers = { db = local.db_name }
  provisioner "local-exec" {
    command = <<-EOT
      CREDS=$(aws secretsmanager get-secret-value --secret-id ${var.db_admin_secret} --query SecretString --output text)
      export PGPASSWORD=$(echo "$CREDS" | jq -r .password)
      PGUSER=$(echo "$CREDS" | jq -r .username)
      psql -h ${var.db_host} -U "$PGUSER" -d postgres \
        -tc "SELECT 1 FROM pg_database WHERE datname = '${local.db_name}'" | grep -q 1 \
        || psql -h ${var.db_host} -U "$PGUSER" -d postgres -c "CREATE DATABASE ${local.db_name}"
    EOT
  }
}

# --- task definition --------------------------------------------------------
resource "aws_ecs_task_definition" "this" {
  family                   = local.name
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 256
  memory                   = 512
  execution_role_arn       = var.execution_role_arn
  task_role_arn            = var.task_role_arn

  container_definitions = jsonencode([{
    name      = "api"
    image     = var.image
    essential = true
    portMappings = [{ containerPort = 8080, protocol = "tcp" }]
    environment = [
      { name = "APP_ENV", value = "preview" },
      { name = "SERVE_STATIC", value = "1" },
    ]
    secrets = [
      # DATABASE_URL assembled at deploy time is simpler for previews:
      # execution role reads the admin secret; the entrypoint gets host/db via env.
      { name = "DATABASE_ADMIN_JSON", valueFrom = var.db_admin_secret },
    ]
    # NOTE: the app reads DATABASE_URL; a thin entrypoint or init step composes
    # it from DATABASE_ADMIN_JSON + DB_HOST + DB_NAME below. Wire this up when
    # implementing (or switch to a per-preview secret).
    logConfiguration = {
      logDriver = "awslogs"
      options = {
        awslogs-group         = "/ecs/${local.name}"
        awslogs-region        = data.aws_region.current.name
        awslogs-stream-prefix = "api"
        awslogs-create-group  = "true"
      }
    }
  }])
}

data "aws_region" "current" {}

# --- service + routing ------------------------------------------------------
resource "aws_lb_target_group" "this" {
  name        = substr(local.name, 0, 32)
  port        = 8080
  protocol    = "HTTP"
  vpc_id      = var.vpc_id
  target_type = "ip"

  health_check {
    path                = "/api/v1/healthz"
    interval            = 15
    healthy_threshold   = 2
    unhealthy_threshold = 3
  }
}

resource "aws_lb_listener_rule" "this" {
  listener_arn = var.listener_arn
  priority     = local.priority

  condition {
    host_header {
      values = [local.host]
    }
  }

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.this.arn
  }
}

resource "aws_ecs_service" "this" {
  name            = local.name
  cluster         = var.cluster_arn
  task_definition = aws_ecs_task_definition.this.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets         = var.private_subnet_ids
    security_groups = [var.api_sg_id]
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.this.arn
    container_name   = "api"
    container_port   = 8080
  }

  depends_on = [aws_lb_listener_rule.this, null_resource.database]
}

resource "aws_route53_record" "this" {
  zone_id = var.hosted_zone_id
  name    = local.host
  type    = "A"

  alias {
    name                   = var.alb_dns_name
    zone_id                = var.alb_zone_id
    evaluate_target_health = false
  }
}
