# base_resource_name is prefixed to the name of every resource created by this module
variable "base_resource_name" {
  type = string
}

# resource_tags are added to all resources that support tags.
variable "resource_tags" {
  type    = map(string)
  default = {}
}

# The name of the Route53 zone that hosts the main .com
variable "buildbeaver_zone_name" {
  type = string
}

variable "app_subdomain" {
  type = string
}

variable "app_lb_dns_name" {
  type = string
}

variable "app_lb_zone_id" {
  type = string
}

variable "runner_subdomain" {
  type = string
}

variable "runner_lb_dns_name" {
  type = string
}

variable "runner_lb_zone_id" {
  type = string
}

variable "app_cloudfront_distribution_domain_name" {
  type = string
}
