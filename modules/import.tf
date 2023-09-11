resource "aws_cloudwatch_log_group" "account_pool_metrics" {
  name              = "/aws/lambda/account_pool_metrics-${var.namespace}"
  retention_in_days = var.cloudwatch_log_retention
}

import {
  to = aws_cloudwatch_log_group.account_pool_metrics
  id = "/aws/lambda/account_pool_metrics-sandbox-20230905"
}

import {
  to = module.lease_auth_lambda.aws_lambda_function.fn
  id = "/aws/lambda/lease_auth-sandbox-20230905"
}

import {
  to = module.credentials_web_page_lambda.aws_lambda_function.fn
  id = "/aws/lambda/credentials_web_page-sandbox-20230905"
}

import {
  to = module.accounts_lambda.aws_lambda_function.fn
  id = "/aws/lambda/accounts-sandbox-20230905"
}

import {
  to = module.leases_lambda.aws_lambda_function.fn
  id = "/aws/lambda/leases_lambda-sandbox-20230905"
}

import {
  to = module.populate_reset_queue.aws_lambda_function.fn
  id = "/aws/lambda/populate_reset_queue-sandbox-20230905"
}

import {
  to = module.process_reset_queue.aws_lambda_function.fn
  id = "/aws/lambda/process_reset_queue-sandbox-20230905"
}

import {
  to = module.fan_out_update_lease_status_lambda.aws_lambda_function.fn
  id = "/aws/lambda/fan_out_update_lease_status-sandbox-20230905"
}

import {
  to = module.update_lease_status_lambda.aws_lambda_function.fn
  id = "/aws/lambda/update_lease_status-sandbox-20230905"
}

import {
  to = module.update_principal_policy.aws_lambda_function.fn
  id = "/aws/lambda/update_principal_policy-sandbox-20230905"
}

import {
  to = module.usage_lambda.aws_lambda_function.fn
  id = "/aws/lambda/usage-sandbox-20230905"
}