# base_resource_name is prefixed to the name of every resource created by this module
variable "base_resource_name" {
  type = string
}

# resource_tags are added to all resources that support tags.
variable "resource_tags" {
  type    = map(string)
  default = {}
}

# The name of the bucket the load balancer will store access logs in
variable "load_balancer_bucket_name" {
  type = string
}
variable "bb_server_bucket_name" {
  type = string
}

variable "frontend_bucket_name" {
  type = string
}

variable "app_cloudfront_distribution_arn" {
  type = string
}

variable "buildbeaver_data_key_arn" {
  type = string
}

variable "buildbeaver_data_key_alias_arn" {
  type = string
}