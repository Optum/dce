locals {
  account_pool_metrics_count = var.account_pool_metrics_toggle == "true" ? 1 : 0
}

module "account_pool_metrics_lambda" {
  source                   = "./lambda"
  name                     = "account_pool_metrics-${var.namespace}"
  namespace                = var.namespace
  description              = "Handles API requests to the /accounts endpoint"
  global_tags              = var.global_tags
  handler                  = "account_pool_metrics"
  alarm_topic_arn          = aws_sns_topic.alarms_topic.arn
  cloudwatch_log_retention = var.cloudwatch_log_retention

  environment = {
    DEBUG              = "false"
    ACCOUNT_ID         = local.account_id
    NAMESPACE          = var.namespace
    AWS_CURRENT_REGION = var.aws_region
    ACCOUNT_DB         = aws_dynamodb_table.accounts.id
  }
}

import {
  to = module.account_pool_metrics_lambda.aws_lambda_function
  id = "/aws/lambda/account_pool_metrics-sandbox-20230905"
}

resource "aws_cloudwatch_event_rule" "every_x_minutes" {
  count               = local.account_pool_metrics_count
  name                = "every-one-minutes"
  description         = "Fires every 1 minutes"
  schedule_expression = var.account_pool_metrics_collection_rate_expression
}

resource "aws_cloudwatch_event_target" "check_account_pool_metrics_at_scheduled_rate" {
  count     = local.account_pool_metrics_count
  rule      = aws_cloudwatch_event_rule.every_x_minutes[0].name
  target_id = "account_pool_metrics_lambda"
  arn       = module.account_pool_metrics_lambda.arn
}

resource "aws_lambda_permission" "allow_cloudwatch_to_call_account_pool_metrics_lambda" {
  count         = local.account_pool_metrics_count
  statement_id  = "AllowExecutionFromCloudWatch"
  action        = "lambda:InvokeFunction"
  function_name = module.account_pool_metrics_lambda.name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.every_x_minutes[0].arn
}

resource "aws_cloudwatch_metric_alarm" "orphaned_accounts" {
  count                     = local.account_pool_metrics_count
  alarm_name                = "orphaned-accounts"
  comparison_operator       = "GreaterThanOrEqualToThreshold"
  evaluation_periods        = "2"
  metric_name               = "OrphanedAccounts"
  namespace                 = local.metrics_namespace
  period                    = "3600"
  statistic                 = "Average"
  threshold                 = var.orphaned_accounts_alarm_threshold
  alarm_description         = "Alarm for orphaned accounts"
  insufficient_data_actions = []
}

resource "aws_cloudwatch_metric_alarm" "too_few_ready_accounts" {
  count                     = local.account_pool_metrics_count
  alarm_name                = "ready-accounts"
  comparison_operator       = "LessThanOrEqualToThreshold"
  evaluation_periods        = "2"
  metric_name               = "ReadyAccounts"
  namespace                 = local.metrics_namespace
  period                    = "3600"
  statistic                 = "Average"
  threshold                 = var.ready_accounts_alarm_threshold
  alarm_description         = "Alarm for too few ready accounts"
  insufficient_data_actions = []
}