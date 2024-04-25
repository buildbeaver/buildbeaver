# base_resource_name is prefixed to the name of every resource created by this module
variable "base_resource_name" {
  type = string
}

# resource_tags are added to all resources that support tags.
variable "resource_tags" {
  type    = map(string)
  default = {}
}

# bb_log_retention_days is the number of days we will hold onto logs in the bb log group
variable "bb_log_retention_days" {
  type = number
}
