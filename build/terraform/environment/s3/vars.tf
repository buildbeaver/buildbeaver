# base_resource_name is prefixed to the name of every resource created by this module
variable "base_resource_name" {
  type = string
}

# resource_tags are added to all resources that support tags.
variable "resource_tags" {
  type    = map(string)
  default = {}
}

variable "app_subdomain" {
  type = string
}

# server_bucket_force_destroy determines if we enable force destroy on our server S3 bucket if it contains data.
# As we want to ensure we do not delete production data, this is set to false by default.
variable "server_bucket_force_destroy" {
  type    = bool
  default = false
}