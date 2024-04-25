# base_resource_name is prefixed to the name of every resource created by this module
variable "base_resource_name" {
  type = string
}

# resource_tags are added to all resources that support tags.
variable "resource_tags" {
  type    = map(string)
  default = {}
}

# The id of the VPC to create the security groups within
variable "vpc_id" {
  type = string
}

variable "app_api_listen_port" {
  type = string
}

variable "runner_api_listen_port" {
  type = string
}

# The port the database listens on
variable "database_port" {
  type = string
}
