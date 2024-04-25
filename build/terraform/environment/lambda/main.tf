terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
    archive = {
      source  = "hashicorp/archive"
      version = "=2.2.0"
    }
  }
}

data "archive_file" "zip" {
  type        = "zip"
  source_file = "${path.module}/index.js"
  output_path = "${path.module}/index.js.zip"
}

resource "aws_lambda_function" "lambda" {
  function_name    = var.base_resource_name
  filename         = data.archive_file.zip.output_path
  source_code_hash = data.archive_file.zip.output_base64sha256
  role             = var.frontend_bucket_lambda_arn
  runtime          = "nodejs16.x"
  handler          = "index.handler"
  publish          = true
}