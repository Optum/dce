resource "aws_lambda_function" "list_dirty_accounts" {
  function_name = "list-dirty-accounts-${var.namespace}"
  handler       = "list_dirty_accounts"
  runtime       = "provided.al2023"
  role          = aws_iam_role.lambda_execution.arn

  filename      = "${path.module}/../../bin/list_dirty_accounts.zip"

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