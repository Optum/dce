module "usage_lambda" {
  source          = "./lambda"
  name            = "usage-${var.namespace}"
  namespace       = var.namespace
  description     = "API /usage endpoints"
  global_tags     = var.global_tags
  handler         = "usage"
  alarm_topic_arn = aws_sns_topic.alarms_topic.arn

  environment = {
    DEBUG              = "false"
    NAMESPACE          = var.namespace
    ACCOUNT_ID         = local.account_id
    AWS_CURRENT_REGION = var.aws_region
    USAGE_CACHE_DB     = aws_dynamodb_table.usage.id
  }
}
