provider "aws" {
  region = var.aws_region
}

# Current AWS Account User
data "aws_caller_identity" "current" {
}

locals {
  account_id = data.aws_caller_identity.current.account_id
}

