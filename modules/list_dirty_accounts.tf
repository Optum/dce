resource "aws_lambda_function" "list_dirty_accounts" {
  function_name = "list_dirty_accounts-${var.namespace}"
  handler       = "list_dirty_accounts"
  runtime       = "provided.al2023" 
  role          = module.list_dirty_accounts_lambda.execution_role_arn
  filename      = "${path.module}/lambda_stub.zip"

  environment {
    variables = {
      ACCOUNT_DB = aws_dynamodb_table.accounts.id
      BUCKET     = aws_s3_bucket.artifacts.id
      S3_KEY     = "dirty_accounts.csv"
      NAMESPACE  = var.namespace
    }
  }

  tags = var.global_tags
}
module "list_dirty_accounts_lambda" {
  source          = "./lambda"
  name            = "list-dirty-accounts-${var.namespace}"
  namespace       = var.namespace
  description     = "Handles API requests to the /list dirty accounts endpoint"
  global_tags     = var.global_tags
  handler         = "list_dirty_accounts"
  alarm_topic_arn = aws_sns_topic.alarms_topic.arn

  environment = {
    DEBUG              = "false"
    BUCKET             = aws_s3_bucket.artifacts.id
    S3_KEY             = "dirty_accounts.csv"
    NAMESPACE          = var.namespace
    AWS_CURRENT_REGION = var.aws_region
    ACCOUNT_DB         = aws_dynamodb_table.accounts.id
  }
}

resource "aws_cloudwatch_event_rule" "list_dirty_accounts_schedule" {
  name                = "list-dirty-accounts-schedule-${var.namespace}"
  description         = "Runs list_dirty_accounts Lambda every Sunday at 3 AM"
  schedule_expression = "cron(0 3 ? * SUN *)"
}

resource "aws_cloudwatch_event_target" "list_dirty_accounts_lambda_target" {
  rule      = aws_cloudwatch_event_rule.list_dirty_accounts_schedule.name
  target_id = "list-dirty-accounts"
  arn       = module.list_dirty_accounts_lambda.arn
}

resource "aws_lambda_permission" "allow_cloudwatch_to_invoke_list_dirty_accounts" {

  statement_id  = "AllowExecutionFromCloudWatch"
  action        = "lambda:InvokeFunction"
  function_name = module.list_dirty_accounts_lambda.name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.list_dirty_accounts_schedule.arn
}
