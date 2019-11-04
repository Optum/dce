# DynamoDB
resource "aws_cloudwatch_metric_alarm" "dynamodb-account-systemerrors-alarm" {
  alarm_name                = "dynamodb-account-systemerrors-alarm-${var.namespace}"
  comparison_operator       = "GreaterThanThreshold"
  evaluation_periods        = 1
  metric_name               = "SystemErrors"
  namespace                 = "AWS/DynamoDB"
  period                    = 60
  statistic                 = "Sum"
  threshold                 = 0
  insufficient_data_actions = []
  alarm_actions             = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    TableName = aws_dynamodb_table.accounts.name
  }
}

resource "aws_cloudwatch_metric_alarm" "dynamodb-lease-systemerrors-alarm" {
  alarm_name                = "dynamodb-lease-systemerrors-alarm-${var.namespace}"
  comparison_operator       = "GreaterThanThreshold"
  evaluation_periods        = 1
  metric_name               = "SystemErrors"
  namespace                 = "AWS/DynamoDB"
  period                    = 60
  statistic                 = "Sum"
  threshold                 = 0
  insufficient_data_actions = []
  alarm_actions             = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    TableName = aws_dynamodb_table.leases.name
  }
}

resource "aws_cloudwatch_metric_alarm" "dynamodb-account-usererrors-alarm" {
  alarm_name                = "dynamodb-account-usererrors-alarm-${var.namespace}"
  comparison_operator       = "GreaterThanThreshold"
  evaluation_periods        = 1
  metric_name               = "UserErrors"
  namespace                 = "AWS/DynamoDB"
  period                    = 60
  statistic                 = "Sum"
  threshold                 = 0
  insufficient_data_actions = []
  alarm_actions             = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    TableName = aws_dynamodb_table.accounts.name
  }
}

resource "aws_cloudwatch_metric_alarm" "dynamodb-lease-usererrors-alarm" {
  alarm_name                = "dynamodb-lease-usererrors-alarm-${var.namespace}"
  comparison_operator       = "GreaterThanThreshold"
  evaluation_periods        = 1
  metric_name               = "UserErrors"
  namespace                 = "AWS/DynamoDB"
  period                    = 60
  statistic                 = "Sum"
  threshold                 = 0
  insufficient_data_actions = []
  alarm_actions             = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    TableName = aws_dynamodb_table.leases.name
  }
}

# API Gateway 4xx errors
resource "aws_cloudwatch_metric_alarm" "apigateway-alarm-4xx" {
  alarm_name                = "apigateway-alarm-4xx-${var.namespace}"
  comparison_operator       = "GreaterThanThreshold"
  evaluation_periods        = 1
  metric_name               = "4XXError"
  namespace                 = "AWS/ApiGateway"
  period                    = 60
  statistic                 = "Sum"
  threshold                 = 50
  insufficient_data_actions = []
  alarm_actions             = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    ApiName = aws_api_gateway_rest_api.gateway_api.name
  }
}

# API Gateway 5xx errors
resource "aws_cloudwatch_metric_alarm" "apigateway-alarm-5xx" {
  alarm_name                = "apigateway-alarm-5xx-${var.namespace}"
  comparison_operator       = "GreaterThanThreshold"
  evaluation_periods        = 1
  metric_name               = "5XXError"
  namespace                 = "AWS/ApiGateway"
  period                    = 60
  statistic                 = "Sum"
  threshold                 = 0
  insufficient_data_actions = []
  alarm_actions             = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    ApiName = aws_api_gateway_rest_api.gateway_api.name
  }
}

resource "aws_cloudwatch_metric_alarm" "apigateway-alarm-latency" {
  alarm_name          = "apigateway-alarm-latency-${var.namespace}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  namespace           = "AWS/ApiGateway"
  metric_name         = "Latency"
  period              = 60
  statistic           = "Sum"
  threshold           = 10000
  alarm_actions       = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    ApiName = aws_api_gateway_rest_api.gateway_api.name
  }
}

resource "aws_cloudwatch_metric_alarm" "apigateway-alarm-integ-latency" {
  alarm_name          = "apigateway-alarm-integ-latency-${var.namespace}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  namespace           = "AWS/ApiGateway"
  metric_name         = "IntegrationLatency"
  period              = 60
  statistic           = "Sum"
  threshold           = 10000
  alarm_actions       = [aws_sns_topic.alarms_topic.arn]

  dimensions = {
    ApiName = aws_api_gateway_rest_api.gateway_api.name
  }
}

# Simple Email Service Alarms
resource "aws_cloudwatch_metric_alarm" "ses-bounce-delivery" {
  alarm_name          = "ses-bounced-delivery-${var.namespace}"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = "2"
  metric_name         = "Reputation.BounceRate"
  namespace           = "AWS/SES"
  period              = "120"
  statistic           = "Maximum"
  threshold           = "1"
  alarm_description   = "This metric monitors email bounce rate"
  alarm_actions       = [aws_sns_topic.alarms_topic.arn]
}

resource "aws_cloudwatch_metric_alarm" "ses-complaint-delivery" {
  alarm_name          = "ses-complaint-delivery-${var.namespace}"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = "2"
  metric_name         = "Reputation.ComplaintRate"
  namespace           = "AWS/SES"
  period              = "120"
  statistic           = "Maximum"
  threshold           = "1"
  alarm_description   = "This metric monitors email complaint rate"
  alarm_actions       = [aws_sns_topic.alarms_topic.arn]
}

resource "aws_cloudwatch_metric_alarm" "ses-reject-delivery" {
  alarm_name          = "ses-reject-delivery-${var.namespace}"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = "2"
  metric_name         = "Reputation.RejectRate"
  namespace           = "AWS/SES"
  period              = "120"
  statistic           = "Maximum"
  threshold           = "1"
  alarm_description   = "This metric monitors email reject rate"
  alarm_actions       = [aws_sns_topic.alarms_topic.arn]
}
