# SQS Queue, for triggering account reset
resource "aws_sqs_queue" "account_reset" {
  name = "account-reset-${var.namespace}"
  tags = var.global_tags
  # Visibility time out should be 6 times the Lambda timeout
  visibility_timeout_seconds = 180
  # A redrive policy that will move messages into a DLQ
  # With 180 second timeout and 40 max recieve count it should give messages
  # up to an hour to work.  The reason this is longer is because
  # of how many concurrent buidlds CodeBuild allows. Lambdas
  # may fail because CodeBuild will no longer take new builds
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.account_reset_dlq.arn
    maxReceiveCount     = 40
  })
}

# SQS Queue, for triggering account reset
resource "aws_sqs_queue" "account_reset_dlq" {
  name                       = "account-reset-dlq-${var.namespace}"
  tags                       = var.global_tags
  visibility_timeout_seconds = 60
}

# Lambda function to add all NotReady accounts to the reset queue
module "populate_reset_queue" {
  source                   = "./lambda"
  name                     = "populate_reset_queue-${var.namespace}"
  namespace                = var.namespace
  description              = "Enqueue all NotReady accounts to be reset."
  global_tags              = var.global_tags
  handler                  = "populate_reset_queue"
  alarm_topic_arn          = aws_sns_topic.alarms_topic.arn
  cloudwatch_log_retention = var.cloudwatch_log_retention

  environment = {
    DEBUG              = "false"
    NAMESPACE          = var.namespace
    ICP_REGION         = var.aws_region
    RESET_SQS_URL      = aws_sqs_queue.account_reset.id
    ACCOUNT_DB         = aws_dynamodb_table.accounts.id
    LEASE_DB           = aws_dynamodb_table.leases.id
    AWS_CURRENT_REGION = var.aws_region
  }
}

# Trigger Global Reset Lambda function on a periodic basis
# https://stackoverflow.com/a/35895316
resource "aws_cloudwatch_event_rule" "populate_reset_queue" {
  name                = "populate-reset-queue-${var.namespace}"
  description         = "Trigger populate_reset_queue Lambda function"
  schedule_expression = var.populate_reset_queue_schedule_expression
}

resource "aws_cloudwatch_event_target" "populate_reset_queue" {
  rule      = aws_cloudwatch_event_rule.populate_reset_queue.name
  target_id = "populate_reset_queue_${var.namespace}"
  arn       = module.populate_reset_queue.arn
}

resource "aws_lambda_permission" "allow_populate_reset_queue" {
  statement_id  = "AllowCloudWatchPopulateResetQueue${title(var.namespace)}"
  action        = "lambda:InvokeFunction"
  function_name = module.populate_reset_queue.name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.populate_reset_queue.arn
}

# Lambda function to execute account reset
# Will poll SQS on a schedule, and execute a CodePipline
# for each account that needs to be reset
module "process_reset_queue" {
  source                   = "./lambda"
  name                     = "process_reset_queue-${var.namespace}"
  namespace                = var.namespace
  description              = "Process events in the reset queue."
  global_tags              = var.global_tags
  handler                  = "process_reset_queue"
  alarm_topic_arn          = aws_sns_topic.alarms_topic.arn
  cloudwatch_log_retention = var.cloudwatch_log_retention

  # Should be a 1/6 of the SQS queue visibility timeout
  timeout = 30

  environment = {
    DEBUG              = "false"
    RESET_BUILD_NAME   = aws_codebuild_project.reset_build.id
    RESET_SQS_URL      = aws_sqs_queue.account_reset.id
    ACCOUNT_DB         = aws_dynamodb_table.accounts.id
    LEASE_DB           = aws_dynamodb_table.leases.id
    AWS_CURRENT_REGION = var.aws_region
  }
}

resource "aws_lambda_event_source_mapping" "process_reset_events_from_sqs" {
  event_source_arn = aws_sqs_queue.account_reset.arn
  function_name    = module.process_reset_queue.arn
  batch_size       = 1
  enabled          = true
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
