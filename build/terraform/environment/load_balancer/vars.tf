# base_resource_name is prefixed to the name of every resource created by this module
variable "base_resource_name" {
  type = string
}

# resource_tags are added to all resources that support tags.
variable "resource_tags" {
  type    = map(string)
  default = {}
}
# The domain name that this environment can be reached on. There must be a certificate
# in the AWS Certificate Manager with a matching domain.
variable "app_lb_certificate_arn" {
  type = string
}

# The id of the VPC to deploy the load balancer to
variable "vpc_id" {
  type = string
}

# A list of subnet ids the load balancer will be routable on
variable "subnet_ids" {
  type = list(any)
}

# The id of the security group to assign to the app load balancer
variable "app_lb_security_group_id" {
  type = string
}

# The App API load balancer terminates SSL for us, so our App API listens on HTTP only.
# This is the port our App API HTTP server can be reached on internally.
variable "app_api_listen_port" {
  type = string
}

# The Runner API load balancer passes through to our Runner API HTTPS server.
variable "runner_api_listen_port" {
  type = string
}

variable "access_logs_bucket_name" {
  type = string
}
