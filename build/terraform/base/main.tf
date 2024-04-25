provider "aws" {
  region = "us-west-2"
}

terraform {
  required_providers {
    random = {
      source  = "hashicorp/random"
      version = "=3.4.3"
    }
    archive = {
      source  = "hashicorp/archive"
      version = "=2.2.0"
    }
  }
  backend "s3" {
    bucket         = "buildbeaver-terraform-state"
    dynamodb_table = "buildbeaver-terraform-state"
    region         = "us-west-2"
  }
}

locals {
  base_resource_name = var.resource_prefix
  resource_tags = {
    Name = local.base_resource_name
  }
}

module "vpc" {
  source             = "./vpc"
  base_resource_name = local.base_resource_name
  resource_tags      = local.resource_tags
}

module "firewall" {
  source             = "./firewall"
  base_resource_name = local.base_resource_name
  resource_tags      = local.resource_tags
  vpc_id             = module.vpc.vpc_id
}
