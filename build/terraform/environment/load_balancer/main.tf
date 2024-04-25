###############################################################################
# App Load Balancer
###############################################################################

resource "random_password" "app_lb_x_bb_secret" {
  length           = 16
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

resource "aws_ssm_parameter" "app_lb_x_bb_secret" {
  name  = "/${var.base_resource_name}/app_lb_x_bb_secret"
  type  = "SecureString"
  value = random_password.app_lb_x_bb_secret.result
}

# Create a public load balancer that we will use to route HTTP/HTTPS traffic to the main API
resource "aws_lb" "app" {
  name               = "${var.base_resource_name}-app"
  internal           = false
  load_balancer_type = "application"
  subnets            = var.subnet_ids
  security_groups    = [var.app_lb_security_group_id]
  access_logs {
    enabled = true
    bucket  = var.access_logs_bucket_name
    prefix  = "app-lb"
  }
  tags = merge(var.resource_tags, {
    Name = "${var.base_resource_name}-app"
  })
}

# Create a target group that specifies the listening port of BuildBeaver on the servers as well as a health check policy
resource "aws_lb_target_group" "app" {
  name                 = "${var.base_resource_name}-app-api"
  port                 = var.app_api_listen_port
  protocol             = "HTTP"
  target_type          = "ip"
  vpc_id               = var.vpc_id
  deregistration_delay = 10
  // TODO add /healthz endpoint and do a proper check
  health_check {
    enabled             = true
    healthy_threshold   = 3
    unhealthy_threshold = 3
    interval            = 10
    timeout             = 5
    matcher             = "200"
    port                = 80
    protocol            = "HTTP"
    path                = "/api/v1"
  }
  lifecycle {
    create_before_destroy = true
  }
}

# Redirect HTTP to HTTPS
resource "aws_lb_listener" "app_http" {
  load_balancer_arn = aws_lb.app.arn
  port              = "80"
  protocol          = "HTTP"
  default_action {
    type = "redirect"
    redirect {
      port        = "443"
      protocol    = "HTTPS"
      status_code = "HTTP_301"
    }
  }
}

# Configure the load balancer to listen on port 443 and forward it to BuildBeaver via the target group above
resource "aws_lb_listener" "app_https" {
  load_balancer_arn = aws_lb.app.arn
  port              = "443"
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-2016-08"
  certificate_arn   = var.app_lb_certificate_arn
  default_action {
    type = "fixed-response"
    fixed_response {
      content_type = "text/plain"
      message_body = "🙊"
      status_code  = "403"
    }
  }
}

resource "aws_lb_listener_rule" "app_https" {
  listener_arn = aws_lb_listener.app_https.arn
  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.app.arn
  }
  condition {
    http_header {
      http_header_name = "X-BB-Secret"
      values           = [aws_ssm_parameter.app_lb_x_bb_secret.value]
    }
  }
}

###############################################################################
# Runner Load Balancer
###############################################################################

resource "aws_lb" "runner" {
  name               = "${var.base_resource_name}-runner"
  internal           = false
  load_balancer_type = "network"
  subnets            = var.subnet_ids
  access_logs {
    enabled = true
    bucket  = var.access_logs_bucket_name
    prefix  = "runner-lb"
  }
  tags = merge(var.resource_tags, {
    Name = "${var.base_resource_name}-runner"
  })
}

resource "aws_lb_target_group" "runner" {
  name                 = "${var.base_resource_name}-runner-api"
  port                 = var.runner_api_listen_port
  protocol             = "TCP"
  target_type          = "ip"
  vpc_id               = var.vpc_id
  deregistration_delay = 10
  # TODO enable
  #  health_check {
  #  }
  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_lb_listener" "runner_https" {
  load_balancer_arn = aws_lb.runner.arn
  port              = "443"
  protocol          = "TCP"
  default_action {
    target_group_arn = aws_lb_target_group.runner.arn
    type             = "forward"
  }
}