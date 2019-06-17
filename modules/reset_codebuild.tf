/**
 * Configure CodePipeline resources to
 * execute our Redbox Account Reset process.
 * - Run aws-nuke in each user account
 * - Re-apply Launchpad to each user account
 *
 * We are currently configured a unique CodePipeline
 * resource for every user account.
 * The Lambda whicxh triggers the CodePipeline refer to them
 * by name, eg. `AccountReset_<AccountId>`
 */

locals {
  # https://stackoverflow.com/a/47243622
  isPr = replace(var.namespace, "pr-", "") != var.namespace
}

# CodeBuild to execute Account Reset
resource "aws_codebuild_project" "reset_build" {
  name          = "redbox-reset-${var.namespace}"
  description   = "Execute Redbox Account reset for an AWS Account"
  build_timeout = "480"
  service_role  = aws_iam_role.codebuild_reset.arn

  source {
    type     = "S3"
    location = "${aws_s3_bucket.artifacts.id}/codebuild/reset.zip"
  }

  artifacts {
    type = "NO_ARTIFACTS"
  }

  environment {
    compute_type                = var.reset_compute_type
    image                       = var.reset_build_image
    type                        = var.reset_build_type
    image_pull_credentials_type = var.reset_image_pull_creds

    environment_variable {
      name  = "RESET_ACCOUNT"
      value = "STUB"
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "RESET_ROLE"
      value = var.reset_assume_role
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "RESET_TEMPLATE"
      value = var.reset_config_template
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "ACCOUNT_DB"
      value = aws_dynamodb_table.redbox_account.id
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "ASSIGNMENT_DB"
      value = aws_dynamodb_table.redbox_account_assignment.id
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "AWS_CURRENT_REGION"
      value = var.aws_region
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "RESET_LAUNCHPAD_BASE_ENDPOINT"
      value = var.reset_launchpad_base_endpoint
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "RESET_LAUNCHPAD_AUTH_ENDPOINT"
      value = var.reset_launchpad_auth_endpoint
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "RESET_LAUNCHPAD_MASTER_ACCOUNT"
      value = var.reset_launchpad_master_account
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "RESET_LAUNCHPAD_BACKEND"
      value = var.reset_launchpad_backend
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "RESET_NUKE_TOGGLE"
      value = var.reset_nuke_toggle // "false" for Dry Run, else Delete Resources
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "RESET_LAUNCHPAD_TOGGLE"
      value = var.reset_launchpad_toggle // "false" for Dry Run, else apply Launchpad
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "RESET_NAMESPACE"
      value = var.namespace
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "IS_PR"
      value = local.isPr ? "true" : "false"
      type  = "PLAINTEXT"
    }
  }

  tags = var.global_tags
}

/**
 * Common Resources,
 * for all account-specific CodePipelines
 */

# Configure an IAM Role for CodeBuild
resource "aws_iam_role" "codebuild_reset" {
  name = "redbox-reset-codebuild-${var.namespace}"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "codebuild.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF


  tags = var.global_tags
}

# Configure IAM Policy for CodeBuild
resource "aws_iam_role_policy" "codebuild_reset" {
  role = aws_iam_role.codebuild_reset.name
  name = "redbox-reset-codebuild-${var.namespace}"

  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Resource": [
        "*"
      ],
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents",
        "sts:AssumeRole",
        "ssm:GetParameter",
        "dynamodb:GetItem",
        "dynamodb:Scan",
        "dynamodb:Query",
        "dynamodb:UpdateItem"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
          "s3:PutObject",
          "s3:GetObject",
          "s3:GetObjectVersion",
          "s3:GetBucketAcl",
          "s3:GetBucketLocation"
      ],
      "Resource": [
        "${aws_s3_bucket.artifacts.arn}",
        "${aws_s3_bucket.artifacts.arn}/*"
      ]
    }
  ]
}
POLICY

}

