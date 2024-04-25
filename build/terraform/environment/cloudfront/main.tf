# Create the cloud front distribution for the naked App domain.
# This domain has two origin servers:
#   The load balancer for the App API is origin for: /api/*
#   The front end s3 bucket is origin for all other requests.
# We additionally configure a Lambda@Edge function to handle the rewrite rules for the React frontend
# app so we can use pretty paths in our URLs (all paths needs to redirect to index.html).
resource "aws_cloudfront_distribution" "frontend" {
  enabled             = true
  is_ipv6_enabled     = true
  aliases             = [var.app_subdomain]
  price_class         = var.price_class
  retain_on_delete    = false
  default_root_object = "index.html"
  origin {
    domain_name              = var.frontend_bucket_regional_domain_name
    origin_id                = "frontend_bucket"
    origin_access_control_id = aws_cloudfront_origin_access_control.frontend_bucket.id
  }
  origin {
    domain_name = var.app_lb_dns_name
    origin_id   = "app_lb"
    custom_origin_config {
      origin_protocol_policy = "https-only"
      http_port              = "80"
      https_port             = "443"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
    custom_header {
      name  = "X-BB-Secret"
      value = var.app_lb_x_bb_secret
    }
  }
  default_cache_behavior {
    target_origin_id       = "frontend_bucket"
    allowed_methods        = ["HEAD", "DELETE", "POST", "GET", "OPTIONS", "PUT", "PATCH"]
    cached_methods         = ["GET", "HEAD"]
    viewer_protocol_policy = "redirect-to-https"
    compress               = true
    min_ttl                = 0
    default_ttl            = 3600
    max_ttl                = 86400
    forwarded_values {
      query_string = false
      cookies {
        forward = "none"
      }
    }
    lambda_function_association {
      event_type   = "origin-request"
      include_body = false
      lambda_arn   = var.frontend_bucket_lambda_arn
    }
  }
  ordered_cache_behavior {
    path_pattern             = "/api/*"
    target_origin_id         = "app_lb"
    allowed_methods          = ["HEAD", "DELETE", "POST", "GET", "OPTIONS", "PUT", "PATCH"]
    cached_methods           = ["GET", "HEAD"]
    viewer_protocol_policy   = "redirect-to-https"
    compress                 = true
    cache_policy_id          = aws_cloudfront_cache_policy.app_api.id
    origin_request_policy_id = aws_cloudfront_origin_request_policy.app_api.id
  }
  viewer_certificate {
    acm_certificate_arn      = var.app_us_east_1_certificate_arn
    ssl_support_method       = "sni-only"
    minimum_protocol_version = "TLSv1.2_2021"
  }
  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }
  tags = var.resource_tags
}

resource "aws_cloudfront_origin_access_control" "frontend_bucket" {
  name                              = "${var.base_resource_name}-frontend-bucket"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

resource "aws_cloudfront_cache_policy" "app_api" {
  name        = var.base_resource_name
  comment     = "Disables caching for the App API"
  default_ttl = 0
  max_ttl     = 0
  min_ttl     = 0
  parameters_in_cache_key_and_forwarded_to_origin {
    cookies_config {
      cookie_behavior = "none"
    }
    headers_config {
      header_behavior = "none"
    }
    query_strings_config {
      query_string_behavior = "none"
    }
  }
}

resource "aws_cloudfront_origin_request_policy" "app_api" {
  name = var.base_resource_name
  headers_config {
    header_behavior = "allViewer"
  }
  cookies_config {
    cookie_behavior = "all"
  }
  query_strings_config {
    query_string_behavior = "all"
  }
}
