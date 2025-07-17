resource "aws_cloudwatch_event_rule" "list_dirty_accounts_schedule" {
  name                = "list_dirty_accounts_every_5_minutes"
  description         = "Triggers the list_dirty_accounts Lambda function every 5 minutes"
  schedule_expression = "rate(5 minutes)" # Runs every 5 minutes
}

resource "aws_cloudwatch_event_target" "list_dirty_accounts_target" {
  rule      = aws_cloudwatch_event_rule.list_dirty_accounts_schedule.name
  target_id = "list_dirty_accounts_lambda_target"
  arn       = aws_lambda_function.list_dirty_accounts.arn
}

resource "aws_lambda_permission" "allow_cloudwatch_to_invoke_list_dirty_accounts" {
  statement_id  = "AllowExecutionFromCloudWatch"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.list_dirty_accounts.function_name
  principal     = "events.amazonaws.com"
}
resource "aws_lambda_function" "list_dirty_accounts" {
  function_name = "list_dirty_accounts"
  description   = "Lambda function to list dirty accounts"
  runtime       = "provided.al2023"
  role          = aws_iam_role.lambda_execution.arn
  handler       = "main"
  filename      = "${path.module}/lambda_stub.zip" # Path to your Lambda deployment package
}