# The database password is expected to have been created previously
data "aws_ssm_parameter" "database_password" {
  name = "/${var.base_resource_name}/database_password"
}

# Create a subnet group that the database will be routable via all of our private subnets
resource "aws_db_subnet_group" "database" {
  name        = var.base_resource_name
  description = "The subnet group to deploy the multi az database to."
  subnet_ids  = var.subnet_ids
  tags        = var.resource_tags
}

# Create the database itself
resource "aws_db_instance" "database" {
  identifier                = var.base_resource_name
  skip_final_snapshot       = var.skip_final_snapshot
  final_snapshot_identifier = "${var.base_resource_name}-final-snapshot-${random_id.final_database_snapshot.hex}"
  allocated_storage         = var.allocated_storage_gb
  engine                    = "postgres"
  engine_version            = "14.4"
  instance_class            = var.server_type
  storage_type              = "gp2"
  multi_az                  = var.multi_az
  publicly_accessible       = false
  backup_retention_period   = var.backup_retention_days
  backup_window             = "10:30-11:30"         # UTC
  maintenance_window        = "Sun:12:30-Sun:13:30" # UTC
  db_name                   = var.database_name
  port                      = var.port
  username                  = var.username
  password                  = data.aws_ssm_parameter.database_password.value
  vpc_security_group_ids    = var.security_group_ids
  db_subnet_group_name      = aws_db_subnet_group.database.name
  copy_tags_to_snapshot     = true
  lifecycle {
    prevent_destroy = false
  }
  tags = var.resource_tags
}

resource "random_id" "final_database_snapshot" {
  byte_length = 16
}
