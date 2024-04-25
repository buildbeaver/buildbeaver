output "bb_server_ecs_task_role_arn" {
  value = aws_iam_role.bb_server_container.arn
}

output "bb_server_ecs_task_execution_role_arn" {
  value = aws_iam_role.bb_server_container_ecs_task_execution.arn
}

output "frontend_lambda_role_arn" {
  value = aws_iam_role.frontend_lambda.arn
}