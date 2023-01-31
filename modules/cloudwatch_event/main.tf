data "aws_arn" "lambda_function" {
  arn = var.lambda_function_arn
}

locals {
  lambda_function_name = split(":", data.aws_arn.lambda_function.resource)[1]
}


resource "aws_cloudwatch_event_rule" "dbbackup" {
  count               = var.enabled ? 1 : 0
  name                = var.name
  description         = var.description
  schedule_expression = var.schedule_expression
}


resource "aws_cloudwatch_event_target" "dbbackup" {
  count     = var.enabled ? 1 : 0
  arn       = var.lambda_function_arn
  rule      = aws_cloudwatch_event_rule.dbbackup[0].name
  target_id = var.name
}

resource "aws_lambda_permission" "dbbackup" {
  count         = var.enabled ? 1 : 0
  statement_id  = "AllowExecutionFromCloudWatch"
  action        = "lambda:InvokeFunction"
  function_name = local.lambda_function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.dbbackup[0].arn
}
