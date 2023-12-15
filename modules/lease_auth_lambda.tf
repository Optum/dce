module "lease_auth_lambda" {
  source          = "./lambda"
  name            = "lease_auth-${var.namespace}"
  namespace       = var.namespace
  description     = "API /leases/id/auth endpoints"
  global_tags     = var.global_tags
  handler         = "lease_auth"
  alarm_topic_arn = aws_sns_topic.alarms_topic.arn

  environment = {
    DEBUG                              = "false"
    NAMESPACE                          = var.namespace
    AWS_CURRENT_REGION                 = var.aws_region
    ACCOUNT_ID                         = local.account_id
    ACCOUNT_DB                         = aws_dynamodb_table.accounts.id
    LEASE_DB                           = aws_dynamodb_table.leases.id
    COGNITO_USER_POOL_ID               = module.api_gateway_authorizer.user_pool_id
    COGNITO_ROLES_ATTRIBUTE_ADMIN_NAME = var.cognito_roles_attribute_admin_name
  }
}
