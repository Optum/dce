module "credentials_web_page_lambda" {
  source          = "./lambda"
  name            = "credentials_web_page-${var.namespace}"
  namespace       = var.namespace
  description     = "Handles API requests to the /credentials_web_page endpoint"
  global_tags     = var.global_tags
  handler         = "credentials_web_page"
  alarm_topic_arn = aws_sns_topic.alarms_topic.arn

  environment = {
    APIGW_DEPLOYMENT_NAME          = "api"
    IDENTITY_POOL_ID               = module.api_gateway_authorizer.identity_pool_id
    SITE_PATH_PREFIX               = "auth"
    USER_POOL_APP_WEB_DOMAIN       = module.api_gateway_authorizer.user_pool_domain
    USER_POOL_CLIENT_ID            = module.api_gateway_authorizer.client_id
    USER_POOL_ID                   = module.api_gateway_authorizer.user_pool_id
    USER_POOL_PROVIDER_NAME        = module.api_gateway_authorizer.user_pool_endpoint
    NAMESPACE                      = var.namespace
    AWS_CURRENT_REGION             = var.aws_region
  }
}