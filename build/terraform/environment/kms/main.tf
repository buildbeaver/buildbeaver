// create a master key that BuildBeaver can use to encrypt all sensitive data
resource "aws_kms_key" "buildbeaver_data_key" {
  description             = "${var.base_resource_name} Data Key"
  deletion_window_in_days = 14
  enable_key_rotation     = false
  tags = merge(var.resource_tags, {
    Name = "${var.base_resource_name}-kms-data-key"
  })
}

// it is good practice to use a key alias - we can change the alias in future to
// point to a new master key if/when we want to manually rotate keys
resource "aws_kms_alias" "buildbeaver_data_key_alias" {
  name          = "alias/${var.base_resource_name}-kms-data-key"
  target_key_id = aws_kms_key.buildbeaver_data_key.key_id
}
