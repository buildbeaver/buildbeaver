output "address" {
  value = aws_db_instance.database.address
}

output "port" {
  value = var.port
}

output "name" {
  value = var.database_name
}

output "username" {
  value = var.username
}