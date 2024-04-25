// TODO autoscaling

data "aws_region" "current" {}

data "aws_ssm_parameter" "database_password" {
  name = "/${var.base_resource_name}/database_password"
}

resource "aws_ssm_parameter" "database_connection_string" {
  name  = "/${var.base_resource_name}/database_connection_string"
  type  = "SecureString"
  value = "${var.bb_server_container_config_database_driver}://${var.bb_server_container_config_database_username}:${data.aws_ssm_parameter.database_password.value}@${var.bb_server_container_config_database_address}:${var.bb_server_container_config_database_port}?sslmode=disable"
}

resource "aws_ecs_cluster" "main" {
  name = var.base_resource_name
  tags = var.resource_tags
}

resource "aws_ecs_task_definition" "main" {
  family                   = var.base_resource_name
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = var.bb_server_container_cpu
  memory                   = var.bb_server_container_memory
  execution_role_arn       = var.bb_server_ecs_task_execution_role_arn
  task_role_arn            = var.bb_server_ecs_task_role_arn
  container_definitions = jsonencode([
    {
      name      = "${var.base_resource_name}-bb-server"
      image     = "${var.bb_server_container_repo}:${var.base_resource_name}-latest"
      essential = true
      portMappings = [
        {
          protocol      = "tcp"
          containerPort = var.bb_server_container_app_api_listen_port
          hostPort      = var.bb_server_container_app_api_listen_port
        },
        {
          protocol      = "tcp"
          containerPort = var.bb_server_container_runner_api_listen_port
          hostPort      = var.bb_server_container_runner_api_listen_port
        }
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = var.bb_server_container_log_group_name
          awslogs-stream-prefix = "bb"
          awslogs-region        = data.aws_region.current.name
        }
      }
      environment = [
        {
          name  = "BB_VAR_database_driver"
          value = var.bb_server_container_config_database_driver
        },
        {
          name  = "BB_VAR_api_server_github_auth_redirect_url"
          value = var.bb_server_container_config_api_server_github_auth_redirect_url
        },
        {
          name  = "BB_VAR_github_app_commit_status_target_url"
          value = var.bb_server_container_config_github_app_commit_status_target_url
        },
        {
          name  = "BB_VAR_blob_store_type"
          value = "AWS_S3"
        },
        {
          name  = "BB_VAR_blob_store_aws_s3_bucket_name"
          value = var.bb_server_container_config_s3_bucket_name
        },
        {
          name  = "BB_VAR_key_manager_type"
          value = "AWS_KMS"
        },
        {
          name  = "BB_VAR_key_manager_aws_kms_master_key_id"
          value = var.bb_server_container_config_kms_master_key_id
        }
      ]
      secrets = [
        {
          name      = "BB_VAR_api_server_session_authentication_key"
          valueFrom = "/${var.base_resource_name}/api_server_session_authentication_key"
        },
        {
          name      = "BB_VAR_api_server_session_encryption_key"
          valueFrom = "/${var.base_resource_name}/api_server_session_encryption_key"
        },
        {
          name      = "BB_VAR_database_connection_string"
          valueFrom = "/${var.base_resource_name}/database_connection_string"
        },
        {
          name      = "BB_VAR_github_app_id"
          valueFrom = "/${var.base_resource_name}/github_app_id"
        },
        {
          name      = "BB_VAR_github_app_private_key"
          valueFrom = "/${var.base_resource_name}/github_app_private_key"
        },
        {
          name      = "BB_VAR_github_client_id"
          valueFrom = "/${var.base_resource_name}/github_client_id"
        },
        {
          name      = "BB_VAR_github_client_secret"
          valueFrom = "/${var.base_resource_name}/github_client_secret"
        },
        {
          name      = "BB_VAR_jwt_certificate_private_key"
          valueFrom = "/${var.base_resource_name}/jwt_certificate_private_key"
        },
        {
          name      = "BB_VAR_jwt_verifying_public_key"
          valueFrom = "/${var.base_resource_name}/jwt_verifying_public_key"
        },
      ]
    }
  ])
  tags = var.resource_tags
}

resource "aws_ecs_service" "main" {
  name                               = var.base_resource_name
  cluster                            = aws_ecs_cluster.main.id
  task_definition                    = aws_ecs_task_definition.main.arn
  desired_count                      = var.bb_server_container_desired_count
  deployment_minimum_healthy_percent = 50
  deployment_maximum_percent         = 200
  health_check_grace_period_seconds  = 60
  launch_type                        = "FARGATE"
  scheduling_strategy                = "REPLICA"
  propagate_tags                     = "SERVICE"
  network_configuration {
    security_groups  = var.bb_server_ecs_task_security_groups
    subnets          = var.bb_server_ecs_task_subnet_ids
    assign_public_ip = false
  }
  # Connect the App API load balancer to the container
  load_balancer {
    target_group_arn = var.bb_server_app_lb_target_group_arn
    container_name   = "${var.base_resource_name}-bb-server"
    container_port   = var.bb_server_container_app_api_listen_port
  }
  # Connect the Runner API load balancer to the container
  load_balancer {
    target_group_arn = var.bb_server_runner_api_lb_target_group_arn
    container_name   = "${var.base_resource_name}-bb-server"
    container_port   = var.bb_server_container_runner_api_listen_port
  }
  # Ignore task_definition changes as the revision changes on deploy of a
  # new version of the images desired_count is ignored as it can change due
  # to autoscaling policy
  lifecycle {
    ignore_changes = [task_definition, desired_count]
  }
  tags = var.resource_tags
}