terraform {
  required_version = "~>0.12.0"
}

provider "aws" {
  region  = var.aws_region
  version = "2.65.0"
}

# Current AWS Account User
data "aws_caller_identity" "current" {
}

locals {
  account_id = data.aws_caller_identity.current.account_id
}

