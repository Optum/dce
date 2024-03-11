terraform {
  # Tested up to 1.7.4
  required_version = ">= 0.13.7"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.26.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

# Current AWS Account User
data "aws_caller_identity" "current" {
}

locals {
  account_id            = data.aws_caller_identity.current.account_id
  sns_encryption_key_id = "alias/aws/sns"
}
