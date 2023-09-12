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

module "api_gateway_authorizer" {
  source             = "./authentication"
  name               = var.namespace_prefix
  namespace          = var.namespace
  callback_urls      = ["${aws_api_gateway_stage.api.invoke_url}/auth"]
  logout_urls        = ["${aws_api_gateway_stage.api.invoke_url}/auth"]
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
