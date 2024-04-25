output "app_cloudfront_distribution_domain_name" {
  value = aws_cloudfront_distribution.frontend.domain_name
}

output "app_cloudfront_distribution_arn" {
  value = aws_cloudfront_distribution.frontend.arn
}