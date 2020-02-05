module "account_pool_metrics_lambda" {
  source          = "./lambda"
  name            = "account_pool_metrics-${var.namespace}"
  namespace       = var.namespace
  description     = "Handles API requests to the /accounts endpoint"
  global_tags     = var.global_tags
  handler         = "account_pool_metrics"
  alarm_topic_arn = aws_sns_topic.alarms_topic.arn

  environment = {
    DEBUG                          = "false"
    ACCOUNT_ID                     = local.account_id
    NAMESPACE                      = var.namespace
    AWS_CURRENT_REGION             = var.aws_region
    ACCOUNT_DB                     = aws_dynamodb_table.accounts.id
  }
}

resource "aws_cloudwatch_event_rule" "every_five_minutes" {
  name = "every-five-minutes"
  description = "Fires every five minutes"
  schedule_expression = "rate(5 minutes)"
}

resource "aws_cloudwatch_event_target" "check_foo_every_five_minutes" {
  rule = aws_cloudwatch_event_rule.every_five_minutes.name
  target_id = "account_pool_metrics_lambda"
  arn = module.account_pool_metrics_lambda.arn
}

resource "aws_lambda_permission" "allow_cloudwatch_to_call_check_foo" {
  statement_id = "AllowExecutionFromCloudWatch"
  action = "lambda:InvokeFunction"
  function_name = module.account_pool_metrics_lambda.name
  principal = "events.amazonaws.com"
  source_arn = aws_cloudwatch_event_rule.every_five_minutes.arn
}