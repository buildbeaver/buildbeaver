# Capture the name of the BuildBeaver (app server) S3 bucket
output "bb_server_bucket_name" {
  value = aws_s3_bucket.bb_server_bucket.id
}

# Capture the name of the load balancer S3 bucket
output "load_balancer_bucket_name" {
  value = aws_s3_bucket.load_balancer_bucket.id
}

output "frontend_bucket_name" {
  value = aws_s3_bucket.frontend.id
}

output "frontend_bucket_regional_domain_name" {
  value = aws_s3_bucket.frontend.bucket_regional_domain_name
}
