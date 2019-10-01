resource "aws_lambda_function" "usage" {
  function_name = "usage-${var.namespace}"
  description   = "API /usage endpoints"
  runtime       = "go1.x"
  handler       = "usage"
  role          = aws_iam_role.redbox_lambda_execution.arn
  timeout       = 300

  environment {
    variables = {
      DEBUG              = "false"
      NAMESPACE          = var.namespace
      AWS_CURRENT_REGION = var.aws_region
      USAGE_CACHE_DB     = aws_dynamodb_table.usage_cache.id
    }
  }

  # Stub an application deployment
  # (deployments will be managed outside terraform)
  filename = data.archive_file.lambda_code_stub.output_path

  lifecycle {
    # Filename will change, as new application deployments are pushed.
    # Prevent terraform from reverting to old application deployments
    # We're not using terraform to manage application deployments
    ignore_changes = [filename]
  }

  tags = var.global_tags
}
