output "frontend_lambda_arn" {
  # Need to export qualified arn here as it includes the Lambda version that CloudFront requires
  value = aws_lambda_function.lambda.qualified_arn
}