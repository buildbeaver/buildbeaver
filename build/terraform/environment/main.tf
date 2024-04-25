provider "aws" {
  region = "us-west-2"
}

# Must be deployed in us-east-1 so it can be used as a Lambda@Edge function with CloudFront
provider "aws" {
  alias  = "aws_east_1"
  region = "us-east-1"
}

terraform {
  required_providers {
    random = {
      source  = "hashicorp/random"
      version = "=3.4.3"
    }
    archive = {
      source  = "hashicorp/archive"
      version = "=2.2.0"
    }
  }
  backend "s3" {
    bucket         = "buildbeaver-terraform-state"
    dynamodb_table = "buildbeaver-terraform-state"
    region         = "us-west-2"
  }
}

locals {
  base_resource_name = "${var.resource_prefix}${var.environment}"
  resource_tags = {
    Name        = local.base_resource_name
    Environment = var.environment
  }
}

module "vpc" {
  source             = "./vpc"
  base_resource_name = local.base_resource_name
  resource_tags      = local.resource_tags
}

module "s3" {
  source                      = "./s3"
  base_resource_name          = local.base_resource_name
  resource_tags               = local.resource_tags
  app_subdomain               = var.dns_app_subdomain
  server_bucket_force_destroy = var.s3_server_bucket_force_destroy
}

module "kms" {
  source             = "./kms"
  base_resource_name = local.base_resource_name
  resource_tags      = local.resource_tags
}

module "iam" {
  source                          = "./iam"
  base_resource_name              = local.base_resource_name
  resource_tags                   = local.resource_tags
  load_balancer_bucket_name       = module.s3.load_balancer_bucket_name
  frontend_bucket_name            = module.s3.frontend_bucket_name
  bb_server_bucket_name          = module.s3.bb_server_bucket_name
  app_cloudfront_distribution_arn = module.cloudfront.app_cloudfront_distribution_arn
  buildbeaver_data_key_arn        = module.kms.buildbeaver_data_key_arn
  buildbeaver_data_key_alias_arn  = module.kms.buildbeaver_data_key_alias_arn
}

module "firewall" {
  source                 = "./firewall"
  base_resource_name     = local.base_resource_name
  resource_tags          = local.resource_tags
  vpc_id                 = module.vpc.vpc_id
  database_port          = var.database_port
  app_api_listen_port    = var.app_api_listen_port
  runner_api_listen_port = var.runner_api_listen_port
}

module "database" {
  source                = "./database"
  base_resource_name    = local.base_resource_name
  resource_tags         = local.resource_tags
  vpc_id                = module.vpc.vpc_id
  subnet_ids            = module.vpc.private_subnet_ids
  security_group_ids    = [module.firewall.private_database_security_group_id]
  port                  = var.database_port
  backup_retention_days = var.database_backup_retention_days
  allocated_storage_gb  = var.database_allocated_storage_gb
  server_type           = var.database_server_type
  multi_az              = var.database_multi_az
  skip_final_snapshot   = var.database_skip_final_snapshot
}

module "load_balancer" {
  source                   = "./load_balancer"
  base_resource_name       = local.base_resource_name
  resource_tags            = local.resource_tags
  vpc_id                   = module.vpc.vpc_id
  subnet_ids               = module.vpc.public_subnet_ids
  app_lb_certificate_arn   = var.app_lb_certificate_arn
  access_logs_bucket_name  = module.s3.load_balancer_bucket_name
  app_lb_security_group_id = module.firewall.public_app_lb_security_group_id
  app_api_listen_port      = var.app_api_listen_port
  runner_api_listen_port   = var.runner_api_listen_port
}

module "cloud_watch" {
  source                 = "./cloudwatch"
  base_resource_name     = local.base_resource_name
  resource_tags          = local.resource_tags
  bb_log_retention_days = var.bb_log_retention_days
}

module "ecs" {
  source                                                          = "./ecs"
  base_resource_name                                              = local.base_resource_name
  resource_tags                                                   = local.resource_tags
  bb_server_container_cpu                                        = var.bb_server_container_cpu
  bb_server_container_memory                                     = var.bb_server_container_memory
  bb_server_container_desired_count                              = var.bb_server_container_desired_count
  bb_server_container_repo                                       = var.bb_server_container_repo
  bb_server_container_log_group_name                             = module.cloud_watch.bb_log_group_name
  bb_server_container_config_api_server_github_auth_redirect_url = var.bb_server_container_config_api_server_github_auth_redirect_url
  bb_server_container_config_github_app_commit_status_target_url = "https://${var.dns_app_subdomain}"
  bb_server_container_app_api_listen_port                        = var.app_api_listen_port
  bb_server_container_runner_api_listen_port                     = var.runner_api_listen_port
  bb_server_container_config_database_address                    = module.database.address
  bb_server_container_config_database_driver                     = var.database_driver
  bb_server_container_config_database_port                       = module.database.port
  bb_server_container_config_database_username                   = module.database.username
  bb_server_container_config_s3_bucket_name                      = module.s3.bb_server_bucket_name
  bb_server_container_config_kms_master_key_id                   = module.kms.buildbeaver_data_key_alias_arn
  bb_server_ecs_task_execution_role_arn                          = module.iam.bb_server_ecs_task_execution_role_arn
  bb_server_ecs_task_role_arn                                    = module.iam.bb_server_ecs_task_role_arn
  bb_server_ecs_task_security_groups                             = [module.firewall.ecs_tasks_security_group_id]
  bb_server_ecs_task_subnet_ids                                  = module.vpc.private_subnet_ids
  bb_server_app_lb_target_group_arn                              = module.load_balancer.app_lb_target_group_arn
  bb_server_runner_api_lb_target_group_arn                       = module.load_balancer.runner_lb_target_group_arn
  depends_on                                                      = [module.load_balancer]
}

module "lambda" {
  providers = {
    aws = aws.aws_east_1
  }
  source                     = "./lambda"
  base_resource_name         = local.base_resource_name
  resource_tags              = local.resource_tags
  frontend_bucket_lambda_arn = module.iam.frontend_lambda_role_arn
}

module "cloudfront" {
  source                               = "./cloudfront"
  base_resource_name                   = local.base_resource_name
  resource_tags                        = local.resource_tags
  app_lb_x_bb_secret                  = module.load_balancer.app_lb_x_bb_secret
  app_subdomain                        = var.dns_app_subdomain
  app_lb_dns_name                      = module.load_balancer.app_lb_dns_name
  app_us_east_1_certificate_arn        = var.app_us_east_1_certificate_arn
  frontend_bucket_lambda_arn           = module.lambda.frontend_lambda_arn
  frontend_bucket_name                 = module.s3.frontend_bucket_name
  frontend_bucket_regional_domain_name = module.s3.frontend_bucket_regional_domain_name
  price_class                          = var.cloud_front_price_class
}

module "dns" {
  source                                  = "./dns"
  base_resource_name                      = local.base_resource_name
  resource_tags                           = local.resource_tags
  buildbeaver_zone_name                   = var.dns_buildbeaver_zone_name
  app_subdomain                           = var.dns_app_subdomain
  app_lb_dns_name                         = module.load_balancer.app_lb_dns_name
  app_lb_zone_id                          = module.load_balancer.app_lb_zone_id
  runner_subdomain                        = var.dns_runner_subdomain
  runner_lb_dns_name                      = module.load_balancer.runner_lb_dns_name
  runner_lb_zone_id                       = module.load_balancer.runner_lb_zone_id
  app_cloudfront_distribution_domain_name = module.cloudfront.app_cloudfront_distribution_domain_name
}
