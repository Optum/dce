output "redbox_account_db_table_name" {
  value = aws_dynamodb_table.redbox_account.name
}

output "redbox_account_assignment_db_table_name" {
  value = aws_dynamodb_table.redbox_account_assignment.name
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

output "dynamodb_table_account_name" {
  value = aws_dynamodb_table.redbox_account.name
}

output "dynamodb_table_account_assignment_name" {
  value = aws_dynamodb_table.redbox_account_assignment.name
}

output "api_url" {
  value = aws_api_gateway_deployment.gateway_deployment_redbox.invoke_url
}

output "alarm_sns_topic_arn" {
  description = "The ARN of the SNS Alarms topic"
  value       = "${aws_sns_topic.alarms_topic.arn}"
}

output "api_access_policy_name" {
  value = aws_iam_policy.api_execute_admin.name
}

output "api_access_policy_arn" {
  value = aws_iam_policy.api_execute_admin.arn
}