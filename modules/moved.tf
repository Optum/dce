moved {
  to = aws_cloudwatch_log_group.account_pool_metrics_lambda
  id = module.account_pool_metrics_lambda.aws_cloudwatch_log_group.fn
}

moved {
  to = aws_cloudwatch_log_group.account_pool_metrics_lambda
  id = module.lease_auth_lambda.aws_cloudwatch_log_group.fn
}

moved {
  to = aws_cloudwatch_log_group.credentials_web_page_lambda
  id = module.credentials_web_page_lambda.aws_cloudwatch_log_group.fn
}

moved {
  to = aws_cloudwatch_log_group.accounts_lambda
  id = module.accounts_lambda.aws_cloudwatch_log_group.fn
}

moved {
  to = aws_cloudwatch_log_group.leases_lambda
  id = module.leases_lambda.aws_cloudwatch_log_group.fn
}

moved {
  to = aws_cloudwatch_log_group.populate_reset_queue
  id = module.populate_reset_queue.aws_cloudwatch_log_group.fn
}

moved {
  to = aws_cloudwatch_log_group.process_reset_queue
  id = module.process_reset_queue.aws_cloudwatch_log_group.fn
}

moved {
  to = aws_cloudwatch_log_group.fan_out_update_lease_status_lambda
  id = module.fan_out_update_lease_status_lambda.aws_cloudwatch_log_group.fn
}

moved {
  to = aws_cloudwatch_log_group.update_lease_status_lambda
  id = module.update_lease_status_lambda.aws_cloudwatch_log_group.fn
}

moved {
  to = aws_cloudwatch_log_group.update_principal_policy
  id = module.update_principal_policy.aws_cloudwatch_log_group.fn
}

moved {
  to = aws_cloudwatch_log_group.usage_lambda
  id = module.usage_lambda.aws_cloudwatch_log_group.fn
}