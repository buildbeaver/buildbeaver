# TODO
#  Need a role + policy for runner servers

###############################################################################
# BB Server Container
###############################################################################

resource "aws_iam_role" "bb_server_container" {
  name = "${var.base_resource_name}-bb-server-ecs-task"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Sid    = ""
        Principal = {
          Service = "ecs-tasks.amazonaws.com"
        }
      },
    ]
  })
  tags = merge(var.resource_tags, {
    Name = "${var.base_resource_name}-bb-server-ecs-task"
  })
}

resource "aws_iam_policy" "bb_server_container" {
  name = "${var.base_resource_name}-bb-server-ecs-task"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      // TODO restrict the allowed S3 ops
      {
        Action = [
          "s3:*",
        ],
        Effect   = "Allow"
        Resource = "arn:aws:s3:::${var.bb_server_bucket_name}"
      },
      {
        Action = [
          "s3:*",
        ],
        Effect   = "Allow"
        Resource = "arn:aws:s3:::${var.bb_server_bucket_name}/*"
      },
      {
        Action = [
          "kms:Encrypt",
          "kms:Decrypt",
          "kms:GenerateDataKey"
        ],
        Effect = "Allow"
        Resource = [
          var.buildbeaver_data_key_arn,
          var.buildbeaver_data_key_alias_arn
        ]
      },
    ]
  })
}

resource "aws_iam_role_policy_attachment" "bb_server_container" {
  role       = aws_iam_role.bb_server_container.name
  policy_arn = aws_iam_policy.bb_server_container.arn
}

###############################################################################
# BB Server Container ECS Task Execution (the role used to bootstrap our ECS containers e.g. pull from our container registry)
###############################################################################

resource "aws_iam_role" "bb_server_container_ecs_task_execution" {
  name = "${var.base_resource_name}-bb-server-container-ecs-task-execution"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Sid    = ""
        Principal = {
          Service = "ecs-tasks.amazonaws.com"
        }
      },
    ]
  })
  tags = merge(var.resource_tags, {
    Name = "${var.base_resource_name}-bb-server-container-ecs-task-execution"
  })
}

resource "aws_iam_policy" "bb_server_container_ecs_task_execution_ecr" {
  name = "${var.base_resource_name}-bb-server-container-ecs-task-execution"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = [
          "ecr:GetAuthorizationToken",
          "ecr:BatchCheckLayerAvailability",
          "ecr:GetDownloadUrlForLayer",
          "ecr:BatchGetImage",
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Effect   = "Allow"
        Resource = "*"
      },
      // TODO limit to specific kms key and ssm params
      {
        Action = [
          "ssm:GetParameters",
          "kms:Decrypt"
        ],
        Effect   = "Allow"
        Resource = "*"
        # TODO
        #        Resource = [
        #          "arn:aws:ssm:<my_region>:<my_account_id>:parameter/SLACK_API_TOKEN",
        #          "arn:aws:km:<my_region>:<my_account_id>:key/<my_key>"
        #        ]
      },
    ]
  })
}

resource "aws_iam_role_policy_attachment" "bb_server_ecs_task_execution_ecr_attachment" {
  role       = aws_iam_role.bb_server_container_ecs_task_execution.name
  policy_arn = aws_iam_policy.bb_server_container_ecs_task_execution_ecr.arn
}

###############################################################################
# Load balancer
###############################################################################

data "aws_iam_policy_document" "load_balancer_policy" {
  # Network Load Balancers
  statement {
    sid    = "AWSLogDeliveryWrite"
    effect = "Allow"
    actions = [
      "s3:PutObject"
    ]
    resources = [
      "arn:aws:s3:::${var.load_balancer_bucket_name}/*"
    ]
    condition {
      test     = "StringEquals"
      variable = "s3:x-amz-acl"
      values = [
        "bucket-owner-full-control"
      ]
    }
    principals {
      type        = "Service"
      identifiers = ["delivery.logs.amazonaws.com"]
    }
  }
  # Network Load Balancers
  statement {
    sid    = "AWSLogDeliveryAclCheck"
    effect = "Allow"
    actions = [
      "s3:GetBucketAcl"
    ]
    resources = [
      "arn:aws:s3:::${var.load_balancer_bucket_name}"
    ]
    principals {
      type        = "Service"
      identifiers = ["delivery.logs.amazonaws.com"]
    }
  }
  # Application load balancers
  statement {
    sid    = ""
    effect = "Allow"
    actions = [
      "s3:PutObject"
    ]
    resources = [
      "arn:aws:s3:::${var.load_balancer_bucket_name}/*"
    ]
    principals {
      type        = "AWS"
      identifiers = ["arn:aws:iam::797873946194:root"]
      # TODO lookup this ID based on region, see https://docs.aws.amazon.com/elasticloadbalancing/latest/application/enable-access-logging.html
    }
  }
}

resource "aws_s3_bucket_policy" "attach_policy_load_balancer_bucket" {
  bucket = var.load_balancer_bucket_name
  policy = data.aws_iam_policy_document.load_balancer_policy.json
}

###############################################################################
# Frontend
###############################################################################

data "aws_iam_policy_document" "frontend_lambda_assume_role" {
  statement {
    sid     = "AllowAwsToAssumeRole"
    effect  = "Allow"
    actions = ["sts:AssumeRole"]
    principals {
      type = "Service"
      identifiers = [
        "edgelambda.amazonaws.com",
        "lambda.amazonaws.com",
      ]
    }
  }
}

resource "aws_iam_role" "frontend_lambda" {
  name               = "${var.base_resource_name}-frontend"
  assume_role_policy = data.aws_iam_policy_document.frontend_lambda_assume_role.json
  tags = merge(var.resource_tags, {
    Name = "${var.base_resource_name}-frontend"
  })
}

data "aws_iam_policy_document" "frontend_lambda" {
  statement {
    effect    = "Allow"
    resources = ["*"]
    actions = [
      "logs:CreateLogStream",
      "logs:PutLogEvents",
      # Lambda@Edge logs are logged into Log Groups in the region of the edge location
      # that executes the code. Because of this, we need to allow the lambda role to create
      # Log Groups in other regions
      # NOTE: We leave this commented out for now as there's currently no way to manage log retention
      # on the log groups that Lambda@Edge creates. If we want to debug the function we'll need to enable
      # this. It's not so bad - our function is so basic the logs aren't particularly useful.
      # "logs:CreateLogGroup",
    ]
  }
}

resource "aws_iam_role_policy" "frontend_lambda" {
  name   = "${var.base_resource_name}-frontend"
  role   = aws_iam_role.frontend_lambda.id
  policy = data.aws_iam_policy_document.frontend_lambda.json
}

data "aws_iam_policy_document" "frontend_bucket" {
  version = "2008-10-17"
  statement {
    sid    = "AllowCloudFrontServicePrincipal"
    effect = "Allow"
    actions = [
      "s3:GetObject",
      "s3:ListBucket"
    ]
    resources = [
      "arn:aws:s3:::${var.frontend_bucket_name}",
      "arn:aws:s3:::${var.frontend_bucket_name}/*"
    ]
    condition {
      test     = "StringEquals"
      variable = "AWS:SourceArn"
      values = [
        var.app_cloudfront_distribution_arn
      ]
    }
    principals {
      identifiers = ["cloudfront.amazonaws.com"]
      type        = "Service"
    }
  }
}

resource "aws_s3_bucket_policy" "attach_policy_frontend_bucket" {
  bucket = var.frontend_bucket_name
  policy = data.aws_iam_policy_document.frontend_bucket.json
}