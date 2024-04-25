# base_resource_name is prefixed to the name of every resource created by this module
variable "base_resource_name" {
  type = string
}

# resource_tags are added to all resources that support tags.
variable "resource_tags" {
  type    = map(string)
  default = {}
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

variable "bb_server_container_app_api_listen_port" {
  type = number
}

variable "bb_server_container_runner_api_listen_port" {
  type = number
}

variable "bb_server_container_log_group_name" {
  type = string
}

variable "bb_server_ecs_task_role_arn" {
  type = string
}

variable "bb_server_ecs_task_execution_role_arn" {
  type = string
}

variable "bb_server_app_lb_target_group_arn" {
  type = string
}

variable "bb_server_runner_api_lb_target_group_arn" {
  type = string
}

variable "bb_server_ecs_task_security_groups" {
  type = list(any)
}

variable "bb_server_ecs_task_subnet_ids" {
  type = list(any)
}

variable "bb_server_container_repo" {
  type = string
}

variable "bb_server_container_config_api_server_github_auth_redirect_url" {
  type = string
}

variable "bb_server_container_config_github_app_commit_status_target_url" {
  type = string
}

variable "bb_server_container_config_database_driver" {
  type = string
}

variable "bb_server_container_config_database_address" {
  type = string
}

variable "bb_server_container_config_database_port" {
  type = string
}

variable "bb_server_container_config_database_username" {
  type = string
}

variable "bb_server_container_config_s3_bucket_name" {
  type = string
}

variable "bb_server_container_config_kms_master_key_id" {
  type = string
}
