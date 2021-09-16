/**
 * Configure CodePipeline resources to
 * execute our Account Reset process.
 * - Run aws-nuke in each user account
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

# CodeBuild to create Azure AD Ent App for AWS Account
# and configure SSO
resource "aws_codebuild_project" "reset_build" {
  name          = "account-reset-${var.namespace}"
  description   = "Reset AWS child accounts"
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
      name = "RESET_ACCOUNT_ADMIN_ROLE_NAME"
      // This value will be passed in by the process_reset_queue
      // lambda, which pulls it from the Accounts DB table
      value = "STUB"
      type  = "PLAINTEXT"
    }

    environment_variable {
      name = "RESET_ACCOUNT_PRINCIPAL_ROLE_NAME"
      // This value will be passed in by the process_reset_queue
      // lambda, which pulls it from the Accounts DB table
      value = "STUB"
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "RESET_ACCOUNT_PRINCIPAL_POLICY_NAME"
      value = local.principal_policy_name
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "RESET_NUKE_TEMPLATE_DEFAULT"
      value = "default-nuke-config-template.yml"
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "RESET_NUKE_TEMPLATE_BUCKET"
      value = var.reset_nuke_template_bucket
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "RESET_NUKE_TEMPLATE_KEY"
      value = var.reset_nuke_template_key
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "ACCOUNT_DB"
      value = aws_dynamodb_table.accounts.id
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "LEASE_DB"
      value = aws_dynamodb_table.leases.id
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "AWS_CURRENT_REGION"
      value = var.aws_region
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "RESET_NUKE_TOGGLE"
      value = var.reset_nuke_toggle // "false" to disable aws-nuke
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "RESET_NUKE_REGIONS"
      value = join(",", var.allowed_regions)
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "RESET_COMPLETE_TOPIC_ARN"
      value = aws_sns_topic.reset_complete.arn
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
  name = "account-reset-codebuild-${var.namespace}"

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

# Configure IAM Role for CodeBuild
resource "aws_iam_role_policy" "codebuild_reset" {
  role   = aws_iam_role.codebuild_reset.name
  name   = "account-reset-codebuild-${var.namespace}"
  policy = data.aws_iam_policy_document.codebuild_reset_policy_document.json
}

# Configure IAM Policy for CodeBuild
data "aws_iam_policy_document" "codebuild_reset_policy_document" {
  override_json = var.reset_nuke_template_bucket == "STUB" ? null : data.aws_iam_policy_document.custom_nuke_config[0].json
  statement {
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents",
      "sts:AssumeRole",
      "ssm:GetParameter",
      "dynamodb:GetItem",
      "dynamodb:Scan",
      "dynamodb:Query",
      "dynamodb:UpdateItem",
      "sns:Publish"
    ]
    resources = ["*"]
  }
  statement {
    actions = [
      "s3:PutObject",
      "s3:GetObject",
      "s3:GetObjectVersion",
      "s3:GetBucketAcl",
      "s3:GetBucketLocation"
    ]
    resources = [
      "${aws_s3_bucket.artifacts.arn}",
      "${aws_s3_bucket.artifacts.arn}/*"
    ]
  }
}

data "aws_iam_policy_document" "custom_nuke_config" {
  count = var.reset_nuke_template_bucket == "STUB" ? 0 : 1
  statement {
    sid = "allowCustomNukeConfig"
    # only put this statement if reset_nuke_template_bucket has been set by the user
    actions = [
      "s3:ListBucket",
      "s3:GetObject"
    ]
    resources = [
      "arn:aws:s3:::${var.reset_nuke_template_bucket}",
      "arn:aws:s3:::${var.reset_nuke_template_bucket}/${var.reset_nuke_template_key}",
    ]
  }
}

# Cloudwatch alarm, for Reset CodeBuild failure
resource "aws_cloudwatch_metric_alarm" "reset_failed_builds" {
  alarm_name = "reset-codebuild-failures-${var.namespace}"

  namespace   = "AWS/CodeBuild"
  metric_name = "FailedBuilds"
  dimensions = {
    ProjectName = aws_codebuild_project.reset_build.name
  }

  # Alarm if 10 or more builds fail within a 5 hour period.
  # Note that resets against `NotReady` accounts run every 6 hours,
  # so this will fail if have 10 accounts that fail to reset in one reset cycle.
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = 5
  threshold           = 10
  period              = 3600
  statistic           = "Sum"

  alarm_actions = [aws_sns_topic.alarms_topic.arn]
}


resource "aws_sns_topic" "reset_complete" {
  name = "account-reset-complete-${var.namespace}"
  tags = var.global_tags
}
