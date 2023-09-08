terraform {
  required_version = ">=0.12.31"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "=3.41.0"
    }

    template = {
      source  = "hashicorp/template"
      version = "2.2.0"
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

  cloudwatch_log_name_prefixes_list_codebuild = [
    "accounts",
  ]

  cloudwatch_log_group_name_prefixes_list_lambda = [
    "account-reset",
    "account_pool_metrics",
    "credentials_web_page",
    "fan_out_update_lease_status",
    "leases",
    "populate_reset_queue",
    "process_reset_queue",
  ]

  cloudwatch_log_groups_list = concat(
    local.cloudwatch_log_name_prefixes_list_codebuild,
    local.cloudwatch_log_group_name_prefixes_list_lambda
  )
  cloudwatch_log_groups_list_lambda    = formatlist("/aws/lambda/%s", local.cloudwatch_log_name_prefixes_list_codebuild)
  cloudwatch_log_groups_list_codebuild = formatlist("/aws/codebuild/%s", local.cloudwatch_log_group_name_prefixes_list_lambda)

  cloudwatch_log_groups_set = toset(
    concat(
      local.cloudwatch_log_groups_list,
      local.cloudwatch_log_groups_list_lambda,
      local.cloudwatch_log_groups_list_codebuild
    )
  )
}
