terraform {
  required_version = "~>0.12.31"
}

provider "aws" {
  region  = var.aws_region
  version = "3.41.0"
}

# Current AWS Account User
data "aws_caller_identity" "current" {
}

locals {
  account_id            = data.aws_caller_identity.current.account_id
  sns_encryption_key_id = "alias/aws/sns"
}

