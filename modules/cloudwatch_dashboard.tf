resource "aws_cloudwatch_dashboard" "main" {
  dashboard_name = "DCE-${var.namespace}"

  dashboard_body = <<EOF
{
  "widgets": [
    {
      "type": "metric",
      "x": 0,
      "y": 0,
      "width": 12,
      "height": 6,
      "properties": {
        "metrics": [
          [ "AWS/Lambda", "Invocations", "FunctionName", "${module.accounts_lambda.name}" ],
          [ ".", "Errors", ".", ".", { "color": "#d62728" } ]
        ],
        "view": "timeSeries",
        "stacked": false,
        "region": "${var.aws_region}",
        "title": "accounts λ",
        "stat": "Sum",
        "period": 300
      }
    },
    {
      "type": "log",
      "x": 12,
      "y": 0,
      "width": 12,
      "height": 6,
      "properties": {
        "query": "SOURCE '/aws/lambda/${module.accounts_lambda.name}' | fields @timestamp, @message\n| sort @timestamp desc\n| limit 100",
        "region": "${var.aws_region}",
        "stacked": false,
        "view": "table"
      }
    },
    {
      "type": "metric",
      "x": 0,
      "y": 6,
      "width": 12,
      "height": 6,
      "properties": {
        "metrics": [
          [ "AWS/Lambda", "Invocations", "FunctionName", "${module.leases_lambda.name}" ],
          [ ".", "Errors", ".", ".", { "color": "#d62728" } ]
        ],
        "view": "timeSeries",
        "stacked": false,
        "region": "${var.aws_region}",
        "title": "leases λ",
        "stat": "Sum",
        "period": 300
      }
    },
    {
      "type": "log",
      "x": 12,
      "y": 6,
      "width": 12,
      "height": 6,
      "properties": {
        "query": "SOURCE '/aws/lambda/${module.leases_lambda.name}' | fields @timestamp, @message\n| sort @timestamp desc\n| limit 100",
        "region": "${var.aws_region}",
        "stacked": false,
        "view": "table"
      }
    },
    {
      "type": "metric",
      "x": 0,
      "y": 6,
      "width": 12,
      "height": 6,
      "properties": {
        "metrics": [
          [ "AWS/Lambda", "Invocations", "FunctionName", "${module.lease_auth_lambda.name}" ],
          [ ".", "Errors", ".", ".", { "color": "#d62728" } ]
        ],
        "view": "timeSeries",
        "stacked": false,
        "region": "${var.aws_region}",
        "title": "lease_auth λ",
        "stat": "Sum",
        "period": 300
      }
    },
    {
      "type": "log",
      "x": 12,
      "y": 6,
      "width": 12,
      "height": 6,
      "properties": {
        "query": "SOURCE '/aws/lambda/${module.lease_auth_lambda.name}' | fields @timestamp, @message\n| sort @timestamp desc\n| limit 100",
        "region": "${var.aws_region}",
        "stacked": false,
        "view": "table"
      }
    },
    {
      "type": "metric",
      "x": 0,
      "y": 12,
      "width": 12,
      "height": 6,
      "properties": {
        "metrics": [
          [ "AWS/Lambda", "Invocations", "FunctionName", "${module.usage_lambda.name}" ],
          [ ".", "Errors", ".", ".", { "color": "#d62728" } ]
        ],
        "view": "timeSeries",
        "stacked": false,
        "region": "${var.aws_region}",
        "title": "usage λ",
        "stat": "Sum",
        "period": 300
      }
    },
    {
      "type": "log",
      "x": 12,
      "y": 12,
      "width": 12,
      "height": 6,
      "properties": {
        "query": "SOURCE '/aws/lambda/${module.usage_lambda.name}' | fields @timestamp, @message\n| sort @timestamp desc\n| limit 100",
        "region": "${var.aws_region}",
        "stacked": false,
        "view": "table"
      }
    },
        {
            "type": "metric",
            "x": 0,
            "y": 24,
            "width": 12,
            "height": 6,
            "properties": {
                "view": "timeSeries",
                "stacked": false,
                "metrics": [
                    [ "AWS/CodeBuild", "SucceededBuilds", "ProjectName", "${aws_codebuild_project.reset_build.name}" ],
                    [ ".", "FailedBuilds", ".", "." ]
                ],
                "region": "${var.aws_region}",
                "period": 300,
                "title": "CodeBuild Resets"
            }
        },
        {
            "type": "log",
            "x": 12,
            "y": 24,
            "width": 6,
            "height": 6,
            "properties": {
                "query": "SOURCE '/aws/codebuild/${aws_codebuild_project.reset_build.name}' | FIELDS @message, @logStream, @timestamp\n| filter @message ~= \": # Child Account\"\n| parse @message \"*: # Child Account\" as account\n| display account, @logStream, @timestamp",
                "region": "${var.aws_region}",
                "stacked": false,
                "view": "table",
                "title": "Recent Resets"
            }
        },
        {
            "type": "log",
            "x": 18,
            "y": 24,
            "width": 6,
            "height": 6,
            "properties": {
                "query": "SOURCE '/aws/codebuild/${aws_codebuild_project.reset_build.name}' | FIELDS @message, @logStream, @timestamp\n| filter @message ~= \": # Child Account\"\n| parse @message \"*: # Child Account\" as account\n| display account, @logStream, @timestamp\n| stats count() by account\n| sort count desc",
                "region": "${var.aws_region}",
                "stacked": false,
                "title": "Most Frequently Reset",
                "view": "table"
            }
        }
  ]
}
 EOF
}