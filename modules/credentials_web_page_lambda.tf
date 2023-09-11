module "credentials_web_page_lambda" {
  source                   = "./lambda"
  name                     = "credentials_web_page-${var.namespace}"
  namespace                = var.namespace
  description              = "Handles API requests to the /credentials_web_page endpoint"
  global_tags              = var.global_tags
  handler                  = "credentials_web_page"
  alarm_topic_arn          = aws_sns_topic.alarms_topic.arn
  cloudwatch_log_retention = var.cloudwatch_log_retention

  environment = {
    APIGW_DEPLOYMENT_NAME       = "api"
    PS_IDENTITY_POOL_ID         = module.ssm_parameter_names.identity_pool_id
    SITE_PATH_PREFIX            = "auth"
    PS_USER_POOL_APP_WEB_DOMAIN = module.ssm_parameter_names.user_pool_domain
    PS_USER_POOL_CLIENT_ID      = module.ssm_parameter_names.client_id
    PS_USER_POOL_ID             = module.ssm_parameter_names.user_pool_id
    PS_USER_POOL_PROVIDER_NAME  = module.ssm_parameter_names.user_pool_endpoint
    NAMESPACE                   = var.namespace
    AWS_CURRENT_REGION          = var.aws_region
  }
}

import {
  to = module.credentials_web_page_lambda.aws_lambda_function
  id = "/aws/lambda/credentials_web_page-sandbox-20230905"
}