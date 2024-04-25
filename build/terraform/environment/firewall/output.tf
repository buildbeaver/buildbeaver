output "public_dmz_security_group_id" {
  value = aws_security_group.dmz.id
}

output "public_app_lb_security_group_id" {
  value = aws_security_group.app_lb.id
}

output "private_database_security_group_id" {
  value = aws_security_group.database.id
}

output "ecs_tasks_security_group_id" {
  value = aws_security_group.bb_server_container.id
}