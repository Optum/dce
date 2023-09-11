module "usage_lambda" {
  source                   = "./lambda"
  name                     = "usage-${var.namespace}"
  namespace                = var.namespace
  description              = "API /usage endpoints"
  global_tags              = var.global_tags
  handler                  = "usage"
  alarm_topic_arn          = aws_sns_topic.alarms_topic.arn
  cloudwatch_log_retention = var.cloudwatch_log_retention

  environment = {
    DEBUG              = "false"
    NAMESPACE          = var.namespace
    AWS_CURRENT_REGION = var.aws_region
    USAGE_CACHE_DB     = aws_dynamodb_table.usage.id
  }
}

import {
  to = module.usage_lambda.aws_lambda_function.fn
  id = "/aws/lambda/usage-sandbox-20230905"
}