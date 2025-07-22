resource "aws_lambda_function" "list_dirty_accounts" {
  function_name = "list-dirty-accounts-${var.namespace}"
  handler       = "list_dirty_accounts"
  runtime       = "provided.al2023"
  role          = module.lambda.execution_role_arn
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

resource "aws_cloudwatch_event_rule" "list_dirty_accounts_schedule" {
  name                = "list-dirty-accounts-schedule-${var.namespace}"
  description         = "Runs list_dirty_accounts Lambda every Sunday at 3 AM"
  schedule_expression = "cron(0 3 ? * SUN *)"
}

resource "aws_cloudwatch_event_target" "list_dirty_accounts_lambda_target" {
  rule      = aws_cloudwatch_event_rule.list_dirty_accounts_schedule.name
  target_id = "list-dirty-accounts"
  arn       = aws_lambda_function.list_dirty_accounts.arn
}

resource "aws_lambda_permission" "allow_cloudwatch_to_invoke_list_dirty_accounts" {

  statement_id  = "AllowExecutionFromCloudWatch"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.list_dirty_accounts.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.list_dirty_accounts_schedule.arn
}
# Lambda function to list dirty accounts and trigger necessary actions
# This function will be scheduled to run weekly to check for dirty accounts
module "lambda" {
  source          = "./lambda"
  namespace       = var.namespace
  name            = "list-dirty-accounts"
  description     = "Lambda function to list dirty accounts"
  handler         = "list_dirty_accounts"
   alarm_topic_arn = aws_sns_topic.alarms_topic.arn
}
