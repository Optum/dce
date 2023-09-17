locals {
  portal_gateway_name = "${var.namespace_prefix}-${var.namespace}"
  stage_name          = "api"
}

resource "aws_api_gateway_rest_api" "gateway_api" {
  name        = local.portal_gateway_name
  description = local.portal_gateway_name
  body        = data.template_file.api_swagger.rendered

  endpoint_configuration {
    types = [var.endpoint_configuration]
  }
}

resource "aws_api_gateway_domain_name" "gateway_api" {
  regional_certificate_arn = data.aws_acm_certificate.custom.arn
  domain_name              = "${var.custom_record_name}.${var.custom_zone_name}"

  endpoint_configuration {
    types = [var.endpoint_configuration]
  }
}

resource "aws_api_gateway_base_path_mapping" "gateway_api_none" {
  api_id      = aws_api_gateway_rest_api.gateway_api.id
  stage_name  = local.stage_name
  domain_name = aws_api_gateway_domain_name.gateway_api.domain_name
}

resource "aws_api_gateway_base_path_mapping" "gateway_api_auth" {
  api_id      = aws_api_gateway_rest_api.gateway_api.id
  stage_name  = local.stage_name
  domain_name = aws_api_gateway_domain_name.gateway_api.domain_name
  base_path   = "auth"
}

module "api_gateway_authorizer" {
  source    = "./authentication"
  name      = var.namespace_prefix
  namespace = var.namespace
  callback_urls = concat(
    ["${aws_api_gateway_stage.api.invoke_url}/auth"],
    var.cognito_callback_urls
  )
  logout_urls = concat(
    ["${aws_api_gateway_stage.api.invoke_url}/auth"],
    var.cognito_logout_urls
  )
  identity_providers = var.cognito_identity_providers
  api_gateway_arn    = aws_api_gateway_stage.api.execution_arn
}

module "ssm_parameter_names" {
  source    = "./ssm_parameter_names"
  namespace = var.namespace
}

resource "aws_ssm_parameter" "identity_pool_id" {
  name  = module.ssm_parameter_names.identity_pool_id
  type  = "String"
  value = module.api_gateway_authorizer.identity_pool_id
}

resource "aws_ssm_parameter" "user_pool_domain" {
  name  = module.ssm_parameter_names.user_pool_domain
  type  = "String"
  value = module.api_gateway_authorizer.user_pool_domain
}

resource "aws_ssm_parameter" "client_id" {
  name  = module.ssm_parameter_names.client_id
  type  = "String"
  value = module.api_gateway_authorizer.client_id
}

resource "aws_ssm_parameter" "user_pool_id" {
  name  = module.ssm_parameter_names.user_pool_id
  type  = "String"
  value = module.api_gateway_authorizer.user_pool_id
}

resource "aws_ssm_parameter" "user_pool_endpoint" {
  name  = module.ssm_parameter_names.user_pool_endpoint
  type  = "String"
  value = module.api_gateway_authorizer.user_pool_endpoint
}

data "template_file" "api_swagger" {
  template = file("${path.module}/swagger.yaml")

  vars = {
    leases_lambda               = module.leases_lambda.invoke_arn
    lease_auth_lambda           = module.lease_auth_lambda.invoke_arn
    accounts_lambda             = module.accounts_lambda.invoke_arn
    usages_lambda               = module.usage_lambda.invoke_arn
    credentials_web_page_lambda = module.credentials_web_page_lambda.invoke_arn
    namespace                   = "${var.namespace_prefix}-${var.namespace}"
  }
}

resource "aws_lambda_permission" "allow_api_gateway" {
  function_name = module.leases_lambda.arn
  statement_id  = "AllowExecutionFromApiGateway"
  action        = "lambda:InvokeFunction"
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.gateway_api.execution_arn}/*/*"
}

resource "aws_lambda_permission" "allow_api_gateway_lease_auth_lambda" {
  function_name = module.lease_auth_lambda.arn
  statement_id  = "AllowExecutionFromApiGateway"
  action        = "lambda:InvokeFunction"
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.gateway_api.execution_arn}/*/*"
}

resource "aws_lambda_permission" "allow_api_gateway_accounts_accounts_lambda" {
  function_name = module.accounts_lambda.arn
  statement_id  = "AllowExecutionFromApiGateway"
  action        = "lambda:InvokeFunction"
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.gateway_api.execution_arn}/*/*"
}

resource "aws_lambda_permission" "allow_api_gateway_usages_lambda" {
  function_name = module.usage_lambda.arn
  statement_id  = "AllowExecutionFromApiGateway"
  action        = "lambda:InvokeFunction"
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.gateway_api.execution_arn}/*/*"
}

resource "aws_lambda_permission" "allow_api_gateway_credentials_web_page_lambda" {
  function_name = module.credentials_web_page_lambda.arn
  statement_id  = "AllowExecutionFromApiGateway"
  action        = "lambda:InvokeFunction"
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.gateway_api.execution_arn}/*/*"
}

resource "aws_api_gateway_stage" "api" {
  stage_name    = local.stage_name
  rest_api_id   = aws_api_gateway_rest_api.gateway_api.id
  deployment_id = aws_api_gateway_deployment.gateway_deployment.id

  xray_tracing_enabled = true

  access_log_settings {
    destination_arn = aws_cloudwatch_log_group.gateway_api_access.arn

    format = jsonencode(
      {
        "caller" : "$context.identity.caller"
        "extendedRequestId" : "$context.extendedRequestId"
        "httpMethod" : "$context.httpMethod"
        "ip" : "$context.identity.sourceIp"
        "protocol" : "$context.protocol"
        "requestId" : "$context.requestId"
        "requestTime" : "$context.requestTime"
        "resourcePath" : "$context.resourcePath"
        "responseLength" : "$context.responseLength"
        "status" : "$context.status"
        "user" : "$context.identity.user"
      }
    )
  }

  depends_on = [aws_cloudwatch_log_group.example]
}

resource "aws_cloudwatch_log_group" "api_gateway_stage" {
  name              = "API-Gateway-Access-Logs_${local.portal_gateway_name}/${local.stage_name}"
  retention_in_days = var.cloudwatch_log_retention
}

resource "aws_api_gateway_deployment" "gateway_deployment" {
  rest_api_id = aws_api_gateway_rest_api.gateway_api.id

  variables = {
    // API Changes won't get deployed, without a trigger in TF
    // See https://medium.com/coryodaniel/til-forcing-terraform-to-deploy-a-aws-api-gateway-deployment-ed36a9f60c1a
    // and https://github.com/terraform-providers/terraform-provider-aws/issues/162#issuecomment-475323730
    change_trigger = sha256(data.template_file.api_swagger.rendered)
  }

  lifecycle {
    create_before_destroy = true
  }
}

// Configure a policy to use for accessing APIs
// This may be consumed by end users, who which to setup
// IAM principals to talk to the APIs
resource "aws_iam_policy" "api_execute_admin" {
  name        = "${var.namespace_prefix}-api-execute-admin-${var.namespace}"
  description = "Provides access to all ${var.namespace_prefix} admin API endpoints"

  policy = <<JSON
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "execute-api:Invoke"
      ],
      "Resource": [
        "${aws_api_gateway_rest_api.gateway_api.execution_arn}/*"
      ]
    }
  ]
}
JSON
}

data "aws_iam_policy_document" "api_gateway_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["apigateway.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "api_gateway_cloudwatch_logs" {
  name                = "dce-api-gateway-cloudwatch-logs"
  assume_role_policy  = data.aws_iam_policy_document.api_gateway_assume_role.json
  managed_policy_arns = [
    "arn:aws:iam::aws:policy/service-role/AmazonAPIGatewayPushToCloudWatchLogs"
  ]
}

resource "aws_api_gateway_account" "cloudwatch_logs" {
  cloudwatch_role_arn = aws_iam_role.api_gateway_cloudwatch_logs.arn
}

resource "aws_cloudwatch_log_group" "gateway_api_execution" {
  name              = "API-Gateway-Execution-Logs_${var.namespace}/${local.stage_name}"
  retention_in_days = 1
}

resource "aws_cloudwatch_log_group" "gateway_api_access" {
  name              = "API-Gateway-Access-Logs_${var.namespace}/${local.stage_name}"
  retention_in_days = 1
}