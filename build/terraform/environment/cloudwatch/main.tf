resource "aws_cloudwatch_log_group" "bb" {
  name              = var.base_resource_name
  retention_in_days = var.bb_log_retention_days
  tags              = var.resource_tags
}
