output "accounts_table_name" {
  value = aws_dynamodb_table.accounts.name
}

output "accounts_table_arn" {
  value = aws_dynamodb_table.accounts.arn
}

output "leases_table_name" {
  value = aws_dynamodb_table.leases.name
}

output "leases_table_arn" {
  value = aws_dynamodb_table.leases.arn
}

output "usage_table_name" {
  value = aws_dynamodb_table.usage.name
}

output "usage_table_arn" {
  value = aws_dynamodb_table.usage.arn
}

output "sqs_reset_queue_url" {
  value = aws_sqs_queue.account_reset.id
}

output "sqs_reset_queue_arn" {
  value = aws_sqs_queue.account_reset.arn
}

output "artifacts_bucket_name" {
  value = aws_s3_bucket.artifacts.id
}

output "artifacts_bucket_arn" {
  value = aws_s3_bucket.artifacts.arn
}

output "namespace" {
  value = var.namespace
}

output "aws_region" {
  value = var.aws_region
}

output "lease_added_topic_id" {
  value = aws_sns_topic.lease_added.id
}

output "lease_added_topic_arn" {
  value = aws_sns_topic.lease_added.arn
}

output "lease_removed_topic_id" {
  value = aws_sns_topic.lease_removed.id
}

output "lease_removed_topic_arn" {
  value = aws_sns_topic.lease_removed.arn
}

output "lease_locked_topic_id" {
  value = aws_sns_topic.lease_locked.id
}

output "lease_locked_topic_arn" {
  value = aws_sns_topic.lease_locked.arn
}

output "lease_unlocked_topic_id" {
  value = aws_sns_topic.lease_unlocked.id
}

output "lease_unlocked_topic_arn" {
  value = aws_sns_topic.lease_unlocked.arn
}

output "reset_complete_topic_arn" {
  value = aws_sns_topic.reset_complete.arn
}

output "account_created_topic_id" {
  value = aws_sns_topic.account_created.id
}

output "account_created_topic_arn" {
  value = aws_sns_topic.account_created.arn
}

output "account_deleted_topic_id" {
  value = aws_sns_topic.account_deleted.id
}

output "account_deleted_topic_arn" {
  value = aws_sns_topic.account_deleted.arn
}

output "api_url" {
  value = aws_api_gateway_stage.api.invoke_url
}

output "alarm_sns_topic_arn" {
  description = "The ARN of the SNS Alarms topic"
  value       = aws_sns_topic.alarms_topic.arn
}

output "api_access_policy_name" {
  value = aws_iam_policy.api_execute_admin.name
}

output "api_access_policy_arn" {
  value = aws_iam_policy.api_execute_admin.arn
}

output "principal_role_name" {
  value = local.principal_role_name
}

output "principal_policy_name" {
  value = local.principal_policy_name
}

output "cognito_user_pool_id" {
  value = module.api_gateway_authorizer.user_pool_id
}

output "cognito_user_pool_client_id" {
  value = module.api_gateway_authorizer.client_id
}

output "cognito_user_pool_endpoint" {
  value = module.api_gateway_authorizer.user_pool_endpoint
}

output "cognito_identity_pool_id" {
  value = module.api_gateway_authorizer.identity_pool_id
}

output "role_user_arn" {
  value = module.api_gateway_authorizer.user_role_arn
}

output "role_admin_arn" {
  value = module.api_gateway_authorizer.admin_role_arn
}

output "role_user_policy" {
  value = module.api_gateway_authorizer.user_policy_arn
}

output "role_admin_policy" {
  value = module.api_gateway_authorizer.admin_policy_arn
}

output "codebuild_reset_name" {
  value = aws_codebuild_project.reset_build.id
}

output "codebuild_reset_role_name" {
  value = aws_iam_role.codebuild_reset.name
}
