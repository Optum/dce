resource "aws_lambda_function" "leases" {
  function_name = "leases-${var.namespace}"
  description   = "API /leases endpoints"
  runtime       = "go1.x"
  handler       = "leases"
  role          = aws_iam_role.redbox_lambda_execution.arn
  timeout       = 300

  environment {
    variables = {
      DEBUG              = "false"
      NAMESPACE          = var.namespace
      AWS_CURRENT_REGION = var.aws_region
      RESET_SQS_URL      = aws_sqs_queue.account_reset.id
      ACCOUNT_DB         = aws_dynamodb_table.redbox_account.id
      LEASE_DB           = aws_dynamodb_table.redbox_lease.id
      PROVISION_TOPIC    = aws_sns_topic.lease_added.arn
      DECOMMISSION_TOPIC = aws_sns_topic.lease_removed.arn
    }
  }

  # Stub an application deployment
  # (deployments will be managed outside terraform)
  filename = data.archive_file.lambda_code_stub.output_path

  lifecycle {
    # Filename will change, as new application deployments are pushed.
    # Prevent terraform from reverting to old application deployments
    # We're not using terraform to manage application deployments
    ignore_changes = [filename]
  }

  tags = var.global_tags
}

resource "aws_sns_topic" "lease_added" {
  name = "lease-added-${var.namespace}"
  tags = var.global_tags
}

resource "aws_sns_topic" "lease_removed" {
  name = "lease-removed-${var.namespace}"
  tags = var.global_tags
}


resource "aws_sns_topic" "lease_locked" {
  name = "lease-locked-${var.namespace}"
  tags = var.global_tags
}

resource "aws_sns_topic" "lease_unlocked" {
  name = "lease-unlocked-${var.namespace}"
  tags = var.global_tags
}