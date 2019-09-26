# SQS Queue, for triggering account reset
resource "aws_sqs_queue" "account_reset" {
  name = "dce-account-reset-${var.namespace}"
  tags = var.global_tags
}

# Lambda function to enqueue all active
# Dce accounts to be reset
# Queries DB for all accounts
# where `status != "READY"`, and adds
# them to an SQS reset queue
resource "aws_lambda_function" "populate_reset_queue" {
  function_name = "populate_reset_queue-${var.namespace}"
  description   = "Enqueue all active Dce accounts to be reset."
  runtime       = "go1.x"
  handler       = "populate_reset_queue"
  role          = aws_iam_role.dce_lambda_execution.arn
  timeout       = 300

  # Stub an application deployment
  # (deployments will be managed outside terraform)
  filename = data.archive_file.lambda_code_stub.output_path

  lifecycle {
    # Filename will change, as new application deployments are pushed.
    # Prevent terraform from reverting to old application deployments
    # We're not using terraform to manage application deployments
    ignore_changes = [filename]
  }

  environment {
    variables = {
      DEBUG              = "false"
      NAMESPACE          = var.namespace
      ICP_REGION         = var.aws_region
      RESET_SQS_URL      = aws_sqs_queue.account_reset.id
      ACCOUNT_DB         = aws_dynamodb_table.dce_account.id
      LEASE_DB           = aws_dynamodb_table.dce_lease.id
      AWS_CURRENT_REGION = var.aws_region
    }
  }

  tags = var.global_tags
}

# Trigger Global Reset Lambda function weekly
# https://stackoverflow.com/a/35895316
resource "aws_cloudwatch_event_rule" "weekly_reset" {
  name                = "dce-weekly-reset-${var.namespace}"
  description         = "Trigger Dce weekly reset"
  schedule_expression = var.weekly_reset_cron_expression
}

resource "aws_cloudwatch_event_target" "weekly_reset" {
  rule      = aws_cloudwatch_event_rule.weekly_reset.name
  target_id = "dce_global_reset_${var.namespace}"
  arn       = aws_lambda_function.populate_reset_queue.arn
}

resource "aws_lambda_permission" "allow_cloudwatch_weekly_reset" {
  statement_id  = "AllowCloudWatchWeeklyReset${title(var.namespace)}"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.populate_reset_queue.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.weekly_reset.arn
}

# Lambda function to execute account reset
# Will poll SQS on a schedule, and execute a CodePipline
# for each account that needs to be reset
resource "aws_lambda_function" "process_reset_queue" {
  function_name = "process_reset_queue-${var.namespace}"
  role          = aws_iam_role.dce_lambda_execution.arn
  handler       = "process_reset_queue"
  runtime       = "go1.x"
  timeout       = 300

  environment {
    variables = {
      DEBUG              = "false"
      RESET_BUILD_NAME   = aws_codebuild_project.reset_build.id
      RESET_SQS_URL      = aws_sqs_queue.account_reset.id
      ACCOUNT_DB         = aws_dynamodb_table.dce_account.id
      LEASE_DB           = aws_dynamodb_table.dce_lease.id
      AWS_CURRENT_REGION = var.aws_region
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

# Trigger Execute Reset lambda function every few minutes
# (to continuously poll SQS reset queue)
resource "aws_cloudwatch_event_rule" "poll_sqs_reset" {
  name                = "dce-poll-reset-queue-${var.namespace}"
  description         = "Poll account reset queue"
  schedule_expression = "rate(3 minutes)"
}

resource "aws_cloudwatch_event_target" "poll_sqs_reset" {
  rule      = aws_cloudwatch_event_rule.poll_sqs_reset.name
  target_id = "dce-poll-reset-queue-${var.namespace}"
  arn       = aws_lambda_function.process_reset_queue.arn
}

resource "aws_lambda_permission" "allow_cloudwatch_poll_sqs_reset" {
  statement_id  = "AllowCloudWatchPollSchedule${title(var.namespace)}"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.process_reset_queue.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.poll_sqs_reset.arn
}

# Lambda code deployments are managed outside of Terraform,
# by our Jenkins pipeline.
# However, Lambda TF resource require a code file to initialize.
# We'll create an empty "stub" file to initialize the lambda function,
# and then publish the code afterwards from jenkins.
data "archive_file" "lambda_code_stub" {
  type        = "zip"
  output_path = "${path.module}/lambda_stub.zip"

  source {
    filename = "stub_file"
    content  = "STUB CONTENT"
  }
}
