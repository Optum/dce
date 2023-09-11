moved {
  from = aws_cloudwatch_log_group.account_pool_metrics_lambda
  to   = module.account_pool_metrics_lambda.aws_cloudwatch_log_group.fn
}

moved {
  from = aws_cloudwatch_log_group.account_pool_metrics_lambda
  to    = module.lease_auth_lambda.aws_cloudwatch_log_group.fn
}

moved {
  from = aws_cloudwatch_log_group.credentials_web_page_lambda
  to    = module.credentials_web_page_lambda.aws_cloudwatch_log_group.fn
}

moved {
  from = aws_cloudwatch_log_group.accounts_lambda
  to    = module.accounts_lambda.aws_cloudwatch_log_group.fn
}

moved {
  from = aws_cloudwatch_log_group.leases_lambda
  to    = module.leases_lambda.aws_cloudwatch_log_group.fn
}

moved {
  from = aws_cloudwatch_log_group.populate_reset_queue
  to    = module.populate_reset_queue.aws_cloudwatch_log_group.fn
}

moved {
  from = aws_cloudwatch_log_group.process_reset_queue
  to    = module.process_reset_queue.aws_cloudwatch_log_group.fn
}

moved {
  from = aws_cloudwatch_log_group.fan_out_update_lease_status_lambda
  to    = module.fan_out_update_lease_status_lambda.aws_cloudwatch_log_group.fn
}

moved {
  from = aws_cloudwatch_log_group.update_lease_status_lambda
  to    = module.update_lease_status_lambda.aws_cloudwatch_log_group.fn
}

moved {
  from = aws_cloudwatch_log_group.update_principal_policy
  to    = module.update_principal_policy.aws_cloudwatch_log_group.fn
}

moved {
  from = aws_cloudwatch_log_group.usage_lambda
  to    = module.usage_lambda.aws_cloudwatch_log_group.fn
}