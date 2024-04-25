# base_resource_name is prefixed to the name of every resource created by this module
variable "base_resource_name" {
  type = string
}

# resource_tags are added to all resources that support tags.
variable "resource_tags" {
  type    = map(string)
  default = {}
}

variable "frontend_bucket_lambda_arn" {
  type = string
}
