###############################################################################
# BB Server
###############################################################################

# Create the bucket that BuildBeaver can store data to
resource "aws_s3_bucket" "bb_server_bucket" {
  bucket        = "${var.base_resource_name}-data"
  force_destroy = var.server_bucket_force_destroy
  tags = merge(var.resource_tags, {
    Name = "${var.base_resource_name}-data"
  })
}

resource "aws_s3_bucket_ownership_controls" "bb_server_bucket_ownership" {
  bucket = aws_s3_bucket.bb_server_bucket.id
  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_s3_bucket_public_access_block" "bb_server_bucket_public_access_block" {
  bucket                  = aws_s3_bucket.bb_server_bucket.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_lifecycle_configuration" "bb_server_bucket_lifecycle" {
  bucket = aws_s3_bucket.bb_server_bucket.id
  # Move artifacts to S3 IA after one week
  rule {
    id     = "archive_blobs"
    status = "Enabled"
    filter {
      prefix = "blobs/"
    }
    transition {
      days          = 30
      storage_class = "STANDARD_IA"
    }
  }
  # Move logs to S3 IA after one week
  rule {
    id     = "archive_logs"
    status = "Enabled"
    filter {
      prefix = "logs/"
    }
    transition {
      days          = 30
      storage_class = "STANDARD_IA"
    }
  }
}

###############################################################################
# App Load Balancer
###############################################################################

# Create the bucket for the load balancers to store access logs in
resource "aws_s3_bucket" "load_balancer_bucket" {
  bucket        = "${var.base_resource_name}-load-balancer"
  force_destroy = true
  tags = merge(var.resource_tags, {
    Name = "${var.base_resource_name}-load-balancer"
  })
}

resource "aws_s3_bucket_ownership_controls" "load_balancer_bucket_ownership" {
  bucket = aws_s3_bucket.load_balancer_bucket.id
  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_s3_bucket_public_access_block" "load_balancer_bucket_public_access_block" {
  bucket                  = aws_s3_bucket.load_balancer_bucket.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

###############################################################################
# Frontend
###############################################################################

# Create an S3 bucket that will hold the frontend static content.
resource "aws_s3_bucket" "frontend" {
  bucket        = var.app_subdomain
  force_destroy = true
  tags = merge(var.resource_tags, {
    Name = "${var.base_resource_name}-frontend"
  })
}

resource "aws_s3_bucket_ownership_controls" "frontend_bucket_ownership" {
  bucket = aws_s3_bucket.frontend.id
  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_s3_bucket_public_access_block" "frontend_bucket_public_access_block" {
  bucket                  = aws_s3_bucket.frontend.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_website_configuration" "frontend" {
  bucket = aws_s3_bucket.frontend.bucket
  index_document {
    suffix = "index.html"
  }
}
