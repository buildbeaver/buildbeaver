# base_resource_name is prefixed to the name of every resource created by this module
variable "base_resource_name" {
  type = string
}

# resource_tags are added to all resources that support tags.
variable "resource_tags" {
  type    = map(string)
  default = {}
}

variable "price_class" {
  type = string
}

variable "app_subdomain" {
  type = string
}

variable "app_lb_dns_name" {
  type = string
}

variable "app_lb_x_bb_secret" {
  type = string
}

variable "frontend_bucket_name" {
  type = string
}

variable "frontend_bucket_regional_domain_name" {
  type = string
}

variable "frontend_bucket_lambda_arn" {
  type = string
}

variable "app_us_east_1_certificate_arn" {
  type = string
}

