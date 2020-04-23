locals {
  metrics_namespace = "DCE/AccountPool"
  vars = {
    api_name                                = aws_api_gateway_rest_api.gateway_api.name
    api_stage_name                          = aws_api_gateway_stage.api.stage_name
    region                                  = var.aws_region
    codebuild_name                          = aws_codebuild_project.reset_build.name
    accounts_lambda_name                    = module.accounts_lambda.name
    leases_lambda_name                      = module.leases_lambda.name
    lease_auth_lambda_name                  = module.lease_auth_lambda.name
    usage_lambda_name                       = module.usage_lambda.name
    populate_reset_queue_name               = module.populate_reset_queue.name
    process_reset_queue_name                = module.process_reset_queue.name
    update_principal_policy_name            = module.update_principal_policy.name
    state_get_lease_name                    = module.state_get_lease_lambda.name
    state_get_lease_usage_name              = module.state_get_lease_usage_lambda.name
    error_scraper_query                     = "fields @timestamp, @message | sort @timestamp desc | filter @message ~= \\\"error\\\" or @message ~= \\\"fail\\\" or @message ~= \\\"fork/exec\\\" | display @timestamp, @message, @logStream| limit 10"
    metrics_namespace_var                   = local.metrics_namespace
    account_pool_metrics_widget_period      = var.account_pool_metrics_widget_period
  }
}

resource "aws_cloudwatch_dashboard" "main" {
  count          = var.cloudwatch_dashboard_toggle == "true" ? 1 : 0
  dashboard_name = "DCE-${var.namespace}"
  dashboard_body = replace(templatefile("${path.module}/fixtures/dashboards/cloudwatch_dashboard.json", local.vars), "$$$$", "$")
}
