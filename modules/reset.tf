# SQS Queue, for triggering account reset
resource "aws_sqs_queue" "account_reset" {
  name = "redbox-account-reset-${var.namespace}"
  tags = var.global_tags
}

# Lambda function to enqueue all active
# Redbox accounts to be reset
# Queries DB for all accounts
# where `status != "READY"`, and adds
# them to an SQS reset queue
module "populate_reset_queue" {
  source          = "./lambda"
  name            = "populate_reset_queue-${var.namespace}"
  namespace       = var.namespace
  description     = "Enqueue all active Redbox accounts to be reset."
  global_tags     = var.global_tags
  handler         = "populate_reset_queue"
  alarm_topic_arn = aws_sns_topic.alarms_topic.arn

  environment = {
    DEBUG              = "false"
    NAMESPACE          = var.namespace
    ICP_REGION         = var.aws_region
    RESET_SQS_URL      = aws_sqs_queue.account_reset.id
    ACCOUNT_DB         = aws_dynamodb_table.redbox_account.id
    LEASE_DB           = aws_dynamodb_table.redbox_lease.id
    AWS_CURRENT_REGION = var.aws_region
  }
}

# Trigger Global Reset Lambda function weekly
# https://stackoverflow.com/a/35895316
resource "aws_cloudwatch_event_rule" "weekly_reset" {
  name                = "redbox-weekly-reset-${var.namespace}"
  description         = "Trigger Redbox weekly reset"
  schedule_expression = var.weekly_reset_cron_expression
}

resource "aws_cloudwatch_event_target" "weekly_reset" {
  rule      = aws_cloudwatch_event_rule.weekly_reset.name
  target_id = "redbox_global_reset_${var.namespace}"
  arn       = module.populate_reset_queue.arn
}

resource "aws_lambda_permission" "allow_cloudwatch_weekly_reset" {
  statement_id  = "AllowCloudWatchWeeklyReset${title(var.namespace)}"
  action        = "lambda:InvokeFunction"
  function_name = module.populate_reset_queue.name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.weekly_reset.arn
}

# Lambda function to execute account reset
# Will poll SQS on a schedule, and execute a CodePipline
# for each account that needs to be reset
module "process_reset_queue" {
  source          = "./lambda"
  name            = "process_reset_queue-${var.namespace}"
  namespace       = var.namespace
  description     = "Process events in the reset queue."
  global_tags     = var.global_tags
  handler         = "process_reset_queue"
  alarm_topic_arn = aws_sns_topic.alarms_topic.arn

  environment = {
    DEBUG              = "false"
    RESET_BUILD_NAME   = aws_codebuild_project.reset_build.id
    RESET_SQS_URL      = aws_sqs_queue.account_reset.id
    ACCOUNT_DB         = aws_dynamodb_table.redbox_account.id
    LEASE_DB           = aws_dynamodb_table.redbox_lease.id
    AWS_CURRENT_REGION = var.aws_region
  }
}

# Trigger Execute Reset lambda function every few minutes
# (to continuously poll SQS reset queue)
resource "aws_cloudwatch_event_rule" "poll_sqs_reset" {
  name                = "redbox-poll-reset-queue-${var.namespace}"
  description         = "Poll account reset queue"
  schedule_expression = "rate(3 minutes)"
}

resource "aws_cloudwatch_event_target" "poll_sqs_reset" {
  rule      = aws_cloudwatch_event_rule.poll_sqs_reset.name
  target_id = "redbox-poll-reset-queue-${var.namespace}"
  arn       = module.process_reset_queue.arn
}

resource "aws_lambda_permission" "allow_cloudwatch_poll_sqs_reset" {
  statement_id  = "AllowCloudWatchPollSchedule${title(var.namespace)}"
  action        = "lambda:InvokeFunction"
  function_name = module.process_reset_queue.name
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
