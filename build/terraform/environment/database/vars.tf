# base_resource_name is prefixed to the name of every resource created by this module
variable "base_resource_name" {
  type = string
}

# resource_tags are added to all resources that support tags.
variable "resource_tags" {
  type    = map(string)
  default = {}
}

# The id of the VPC to deploy the database to
variable "vpc_id" {
  type = string
}

# A list of subnet ids the database will be routable on
variable "subnet_ids" {
  type = list(any)
}

# The ids of the security groups to assign to the database
variable "security_group_ids" {
  type = list(any)
}

# How long to retain database backups for
variable "backup_retention_days" {
  type = string
}

# The amount of space in GB that will be provisioned for the database
variable "allocated_storage_gb" {
  type = string
}

# The port the Postgres database will listen on
variable "port" {
  type = string
}

# The username we will configure the Postgres database to recognize
variable "username" {
  type    = string
  default = "buildbeaver"
}

# The name of the database inside Postgres that we will use
variable "database_name" {
  type    = string
  default = "buildbeaver"
}

# The type of server to run the database on
variable "server_type" {
  type = string
}

# True if the database should have multi-az replication and fail over enabled
variable "multi_az" {
  type = bool
}

# True if a final database snapshot should not be taken when deleting the database (should always be false for prod
# and staging)
variable "skip_final_snapshot" {
  type = bool
}