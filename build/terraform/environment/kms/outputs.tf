output "buildbeaver_data_key_arn" {
  value = aws_kms_key.buildbeaver_data_key.arn
}

output "buildbeaver_data_key_alias_arn" {
  value = aws_kms_alias.buildbeaver_data_key_alias.arn
}