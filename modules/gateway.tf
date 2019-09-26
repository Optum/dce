locals {
  portal_gateway_name = "AWS_Dce-${var.namespace}"
}

resource "aws_api_gateway_rest_api" "gateway_api" {
  name        = local.portal_gateway_name
  description = local.portal_gateway_name
  body        = data.template_file.aws_dce_api_swagger.rendered
}

data "template_file" "aws_dce_api_swagger" {
  template = file("${path.module}/swaggerDce.yaml")

  vars = {
    router_lambda_arn = aws_lambda_function.leases.invoke_arn
    accounts_lambda   = aws_lambda_function.accounts_lambda.invoke_arn
    namespace         = "AWS_Dce-${var.namespace}"
  }
}

resource "aws_lambda_permission" "allow_api_gateway" {
  function_name = aws_lambda_function.leases.arn
  statement_id  = "AllowExecutionFromApiGateway"
  action        = "lambda:InvokeFunction"
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.gateway_api.execution_arn}/*/*"
}

resource "aws_lambda_permission" "allow_api_gateway_accounts_accounts_lambda" {
  function_name = aws_lambda_function.accounts_lambda.arn
  statement_id  = "AllowExecutionFromApiGateway"
  action        = "lambda:InvokeFunction"
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.gateway_api.execution_arn}/*/*"
}

resource "aws_api_gateway_deployment" "gateway_deployment_dce" {
  rest_api_id = aws_api_gateway_rest_api.gateway_api.id

  stage_name = "dce-${var.namespace}"

  variables = {
    // API Changes won't get deployed, without a trigger in TF
    // See https://medium.com/coryodaniel/til-forcing-terraform-to-deploy-a-aws-api-gateway-deployment-ed36a9f60c1a
    // and https://github.com/terraform-providers/terraform-provider-aws/issues/162#issuecomment-475323730
    change_trigger = sha256(data.template_file.aws_dce_api_swagger.rendered)
  }

  lifecycle {
    create_before_destroy = true
  }
}

// Configure a policy to use for accessing APIs
// This may be consumed by end users, who which to setup
// IAM principals to talk to Dce APIs
resource "aws_iam_policy" "api_execute_admin" {
  name        = "dce-api-execute-admin-${var.namespace}"
  description = "Provides access to all Dce admin API endpoints"

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
