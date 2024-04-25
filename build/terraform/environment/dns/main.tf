# http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-route53-aliastarget.html
variable "cf_alias_zone_id" {
  description = "Fixed hardcoded constant zone_id that is used for all CloudFront distributions"
  default     = "Z2FDTNDATAQYW2"
}

# The primary zone is expected to exist already
data "aws_route53_zone" "primary" {
  name = var.buildbeaver_zone_name
}

# Create an A record that aliases the cloudfront distribution used to serve both the frontend and our app api
resource "aws_route53_record" "app" {
  zone_id = data.aws_route53_zone.primary.zone_id
  name    = var.app_subdomain
  type    = "A"
  alias {
    name                   = var.app_cloudfront_distribution_domain_name
    zone_id                = var.cf_alias_zone_id
    evaluate_target_health = true
  }
}

# Create an A record that aliases the load balancer used to deliver traffic to our app servers.
resource "aws_route53_record" "runner" {
  zone_id = data.aws_route53_zone.primary.zone_id
  name    = var.runner_subdomain
  type    = "A"
  alias {
    name                   = var.runner_lb_dns_name
    zone_id                = var.runner_lb_zone_id
    evaluate_target_health = true
  }
}
