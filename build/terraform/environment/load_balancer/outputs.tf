output "app_lb_target_group_arn" {
  value = aws_lb_target_group.app.arn
}

output "app_lb_dns_name" {
  value = aws_lb.app.dns_name
}

output "app_lb_zone_id" {
  value = aws_lb.app.zone_id
}

output "app_lb_x_bb_secret" {
  value = aws_ssm_parameter.app_lb_x_bb_secret.value
}

output "runner_lb_target_group_arn" {
  value = aws_lb_target_group.runner.arn
}

output "runner_lb_dns_name" {
  value = aws_lb.runner.dns_name
}

output "runner_lb_zone_id" {
  value = aws_lb.runner.zone_id
}

