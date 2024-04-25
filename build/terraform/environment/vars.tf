###############################################################################
# Generic
###############################################################################

variable "resource_prefix" {
  default = "buildbeaver-"
}

variable "environment" {
  type = string
}

###############################################################################
# Database
###############################################################################

variable "database_driver" {
  default = "postgres"
}

variable "database_port" {
  default = "5432"
}

variable "database_backup_retention_days" {
  type = number
}

variable "database_allocated_storage_gb" {
  type = number
}

variable "database_server_type" {
  type = string
}

variable "database_multi_az" {
  type = bool
}

# True if a final database snapshot should not be taken when deleting the database (should always be false for prod
# and staging)
variable "database_skip_final_snapshot" {
  type = bool
}

###############################################################################
# DNS
###############################################################################

# TODO: Replace the API endpoints with the domain your instance is running on
variable "dns_buildbeaver_zone_name" {
  default = "changeme.com."
}

variable "dns_app_subdomain" {
  type = string
}

variable "dns_runner_subdomain" {
  type = string
}

###############################################################################
# Static Hosting
###############################################################################

variable "cloud_front_price_class" {
  type = string
}

###############################################################################
# Networking
###############################################################################

variable "app_lb_certificate_arn" {
  type = string
}

variable "app_us_east_1_certificate_arn" {
  type = string
}

variable "app_api_listen_port" {
  default = "80"
}

variable "runner_api_listen_port" {
  default = "443"
}

###############################################################################
# CloudWatch
###############################################################################

variable "bb_log_retention_days" {
  type = number
}

###############################################################################
# BB Server Containers
###############################################################################

variable "bb_server_container_repo" {
  type = string
}

variable "bb_server_container_cpu" {
  type = number
}

variable "bb_server_container_memory" {
  type = number
}

variable "bb_server_container_desired_count" {
  type = number
}

variable "bb_server_container_config_api_server_github_auth_redirect_url" {
  type = string
}

###############################################################################
# S3
###############################################################################

variable "s3_server_bucket_force_destroy" {
  type    = bool
  default = false
}