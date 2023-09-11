module "lease_auth_lambda" {
  source                   = "./lambda"
  name                     = "lease_auth-${var.namespace}"
  namespace                = var.namespace
  description              = "API /leases/id/auth endpoints"
  global_tags              = var.global_tags
  handler                  = "lease_auth"
  alarm_topic_arn          = aws_sns_topic.alarms_topic.arn
  cloudwatch_log_retention = var.cloudwatch_log_retention

  environment = {
    DEBUG                              = "false"
    NAMESPACE                          = var.namespace
    AWS_CURRENT_REGION                 = var.aws_region
    ACCOUNT_DB                         = aws_dynamodb_table.accounts.id
    LEASE_DB                           = aws_dynamodb_table.leases.id
    COGNITO_USER_POOL_ID               = module.api_gateway_authorizer.user_pool_id
    COGNITO_ROLES_ATTRIBUTE_ADMIN_NAME = var.cognito_roles_attribute_admin_name
  }
}

# import {
#   to = module.lease_auth_lambda.aws_lambda_function.fn
#   id = "/aws/lambda/lease_auth-sandbox-20230905"
# }