resource "aws_cloudwatch_dashboard" "main" {
  count          = var.cloudwatch_dashboard_toggle == "true" ? 1 : 0
  dashboard_name = "DCE-${var.namespace}"

  dashboard_body = <<EOF
{
  "widgets": [
    {
        "type": "metric",
        "x": 0,
        "y": 0,
        "width": 24,
        "height": 6,
        "properties": {
            "metrics": [
                [ "AWS/ApiGateway", "5XXError", "ApiName", "${aws_api_gateway_rest_api.gateway_api.name}", "Stage", "${aws_api_gateway_stage.api.stage_name}", { "color": "#d62728" } ],
                [ ".", "4XXError", ".", ".", ".", "." ]
            ],
            "view": "timeSeries",
            "stacked": false,
            "title": "API Errors",
            "region": "${var.aws_region}",
            "legend": {
                "position": "bottom"
            },
            "stat": "Average",
            "period": 300
        }
    },
    {
        "type": "metric",
        "x": 0,
        "y": 6,
        "width": 12,
        "height": 6,
        "properties": {
            "view": "timeSeries",
            "stacked": false,
            "metrics": [
                [ "AWS/CodeBuild", "SucceededBuilds", "ProjectName", "${aws_codebuild_project.reset_build.name}", { "color": "#1f77b4" } ],
                [ ".", "FailedBuilds", ".", ".", { "color": "#d62728" } ]
            ],
            "region": "${var.aws_region}",
            "period": 300,
            "title": "resets codebuild"
        }
    },
    {
        "type": "log",
        "x": 12,
        "y": 6,
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
        "y": 6,
        "width": 6,
        "height": 6,
        "properties": {
            "query": "SOURCE '/aws/codebuild/${aws_codebuild_project.reset_build.name}' | FIELDS @message, @logStream, @timestamp\n| filter @message ~= \": # Child Account\"\n| parse @message \"*: # Child Account\" as account\n| display account, @logStream, @timestamp\n| stats count() by account\n| sort count desc",
            "region": "${var.aws_region}",
            "stacked": false,
            "title": "Most Frequently Reset",
            "view": "table"
        }
    },
    {
        "type": "log",
        "x": 0,
        "y": 12,
        "width": 24,
        "height": 6,
        "properties": {
            "query": "SOURCE '/aws/codebuild/${aws_codebuild_project.reset_build.name}' | fields @timestamp, @message\n| sort @timestamp desc\n| filter @message ~= \"error\"\n| display @timestamp, @message, @logStream\n| limit 100\n",
            "region": "us-east-1",
            "stacked": false,
            "view": "table",
            "title": "resets codebuild error scraper"
        }
    },
    {
      "type": "metric",
      "x": 0,
      "y": 18,
      "width": 12,
      "height": 6,
      "properties": {
        "metrics": [
          [ "AWS/Lambda", "Invocations", "FunctionName", "${module.accounts_lambda.name}", { "color": "#1f77b4" } ],
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
      "y": 18,
      "width": 12,
      "height": 6,
      "properties": {
        "query": "SOURCE '/aws/lambda/${module.accounts_lambda.name}' | fields @timestamp, @message\n| sort @timestamp desc\n| filter @message ~= \"error\"\n| display @timestamp, @message, @logStream\n| limit 100\n",
        "region": "${var.aws_region}",
        "stacked": false,
        "view": "table",
        "title": "accounts λ error scraper"
      }
    },
    {
      "type": "metric",
      "x": 0,
      "y": 24,
      "width": 12,
      "height": 6,
      "properties": {
        "metrics": [
          [ "AWS/Lambda", "Invocations", "FunctionName", "${module.leases_lambda.name}", { "color": "#1f77b4" } ],
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
      "y": 24,
      "width": 12,
      "height": 6,
      "properties": {
        "query": "SOURCE '/aws/lambda/${module.leases_lambda.name}' | fields @timestamp, @message\n| sort @timestamp desc\n| filter @message ~= \"error\"\n| display @timestamp, @message, @logStream\n| limit 100\n",
        "region": "${var.aws_region}",
        "stacked": false,
        "view": "table",
        "title": "leases λ error scraper"
      }
    },
    {
      "type": "metric",
      "x": 0,
      "y": 30,
      "width": 12,
      "height": 6,
      "properties": {
        "metrics": [
          [ "AWS/Lambda", "Invocations", "FunctionName", "${module.lease_auth_lambda.name}", { "color": "#1f77b4" } ],
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
      "y": 30,
      "width": 12,
      "height": 6,
      "properties": {
        "query": "SOURCE '/aws/lambda/${module.lease_auth_lambda.name}' | fields @timestamp, @message\n| sort @timestamp desc\n| filter @message ~= \"error\"\n| display @timestamp, @message, @logStream\n| limit 100\n",
        "region": "${var.aws_region}",
        "stacked": false,
        "view": "table",
        "title": "lease_auth λ error scraper"
      }
    },
    {
      "type": "metric",
      "x": 0,
      "y": 36,
      "width": 12,
      "height": 6,
      "properties": {
        "metrics": [
          [ "AWS/Lambda", "Invocations", "FunctionName", "${module.usage_lambda.name}", { "color": "#1f77b4" } ],
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
      "y": 36,
      "width": 12,
      "height": 6,
      "properties": {
        "query": "SOURCE '/aws/lambda/${module.usage_lambda.name}' | fields @timestamp, @message\n| sort @timestamp desc\n| filter @message ~= \"error\"\n| display @timestamp, @message, @logStream\n| limit 100\n",
        "region": "${var.aws_region}",
        "stacked": false,
        "view": "table",
        "title": "usage λ error scraper"
      }
    },
    {
        "type": "metric",
        "x": 0,
        "y": 42,
        "width": 12,
        "height": 6,
        "properties": {
            "view": "timeSeries",
            "stacked": false,
            "metrics": [
                [ "AWS/Lambda", "Invocations", "FunctionName", "${module.update_lease_status_lambda.name}", { "color": "#1f77b4" } ],
                [ ".", "Errors", ".", ".", { "color": "#d62728" } ]
            ],
            "region": "${var.aws_region}",
            "title": "${module.update_lease_status_lambda.name} λ"
        }
    },
    {
      "type": "log",
      "x": 12,
      "y": 42,
      "width": 12,
      "height": 6,
      "properties": {
        "query": "SOURCE '/aws/lambda/${module.update_lease_status_lambda.name}' | fields @timestamp, @message\n| sort @timestamp desc\n| filter @message ~= \"error\"\n| display @timestamp, @message, @logStream\n| limit 100\n",
        "region": "${var.aws_region}",
        "stacked": false,
        "view": "table",
        "title": "${module.update_lease_status_lambda.name} λ error scraper"
      }
    },
    {
        "type": "metric",
        "x": 0,
        "y": 48,
        "width": 12,
        "height": 6,
        "properties": {
            "view": "timeSeries",
            "stacked": false,
            "metrics": [
                [ "AWS/Lambda", "Invocations", "FunctionName", "${module.fan_out_update_lease_status_lambda.name}", { "color": "#1f77b4" } ],
                [ ".", "Errors", ".", ".", { "color": "#d62728" } ]
            ],
            "region": "${var.aws_region}",
            "title": "${module.fan_out_update_lease_status_lambda.name} λ"
        }
    },
    {
      "type": "log",
      "x": 12,
      "y": 48,
      "width": 12,
      "height": 6,
      "properties": {
        "query": "SOURCE '/aws/lambda/${module.fan_out_update_lease_status_lambda.name}' | fields @timestamp, @message\n| sort @timestamp desc\n| filter @message ~= \"error\"\n| display @timestamp, @message, @logStream\n| limit 100\n",
        "region": "${var.aws_region}",
        "stacked": false,
        "view": "table",
        "title": "${module.fan_out_update_lease_status_lambda.name} λ error scraper"
      }
    },
    {
        "type": "metric",
        "x": 0,
        "y": 54,
        "width": 12,
        "height": 6,
        "properties": {
            "view": "timeSeries",
            "stacked": false,
            "metrics": [
                [ "AWS/Lambda", "Invocations", "FunctionName", "${module.populate_reset_queue.name}", { "color": "#1f77b4" } ],
                [ ".", "Errors", ".", ".", { "color": "#d62728" } ]
            ],
            "region": "${var.aws_region}",
            "title": "${module.populate_reset_queue.name} λ"
        }
    },
    {
      "type": "log",
      "x": 12,
      "y": 54,
      "width": 12,
      "height": 6,
      "properties": {
        "query": "SOURCE '/aws/lambda/${module.populate_reset_queue.name}' | fields @timestamp, @message\n| sort @timestamp desc\n| filter @message ~= \"error\"\n| display @timestamp, @message, @logStream\n| limit 100\n",
        "region": "${var.aws_region}",
        "stacked": false,
        "view": "table",
        "title": "${module.populate_reset_queue.name} λ error scraper"
      }
    },
    {
        "type": "metric",
        "x": 0,
        "y": 60,
        "width": 12,
        "height": 6,
        "properties": {
            "view": "timeSeries",
            "stacked": false,
            "metrics": [
                [ "AWS/Lambda", "Invocations", "FunctionName", "${module.process_reset_queue.name}", { "color": "#1f77b4" } ],
                [ ".", "Errors", ".", ".", { "color": "#d62728" } ]
            ],
            "region": "${var.aws_region}",
            "title": "${module.process_reset_queue.name} λ"
        }
    },
    {
      "type": "log",
      "x": 12,
      "y": 60,
      "width": 12,
      "height": 6,
      "properties": {
        "query": "SOURCE '/aws/lambda/${module.process_reset_queue.name}' | fields @timestamp, @message\n| sort @timestamp desc\n| filter @message ~= \"error\"\n| display @timestamp, @message, @logStream\n| limit 100\n",
        "region": "${var.aws_region}",
        "stacked": false,
        "view": "table",
        "title": "${module.process_reset_queue.name} λ error scraper"
      }
    },
    {
        "type": "metric",
        "x": 0,
        "y": 66,
        "width": 12,
        "height": 6,
        "properties": {
            "view": "timeSeries",
            "stacked": false,
            "metrics": [
                [ "AWS/Lambda", "Invocations", "FunctionName", "${module.publish_lease_events_lambda.name}", { "color": "#1f77b4" } ],
                [ ".", "Errors", ".", ".", { "color": "#d62728" } ]
            ],
            "region": "${var.aws_region}",
            "title": "${module.publish_lease_events_lambda.name} λ"
        }
    },
    {
      "type": "log",
      "x": 12,
      "y": 66,
      "width": 12,
      "height": 6,
      "properties": {
        "query": "SOURCE '/aws/lambda/${module.publish_lease_events_lambda.name}' | fields @timestamp, @message\n| sort @timestamp desc\n| filter @message ~= \"error\"\n| display @timestamp, @message, @logStream\n| limit 100\n",
        "region": "${var.aws_region}",
        "stacked": false,
        "view": "table",
        "title": "${module.publish_lease_events_lambda.name} λ error scraper"
      }
    },
    {
        "type": "metric",
        "x": 0,
        "y": 72,
        "width": 12,
        "height": 6,
        "properties": {
            "view": "timeSeries",
            "stacked": false,
            "metrics": [
                [ "AWS/Lambda", "Invocations", "FunctionName", "${module.update_lease_status_lambda.name}", { "color": "#1f77b4" } ],
                [ ".", "Errors", ".", ".", { "color": "#d62728" } ]
            ],
            "region": "${var.aws_region}",
            "title": "${module.update_lease_status_lambda.name} λ"
        }
    },
    {
      "type": "log",
      "x": 12,
      "y": 72,
      "width": 12,
      "height": 6,
      "properties": {
        "query": "SOURCE '/aws/lambda/${module.update_lease_status_lambda.name}' | fields @timestamp, @message\n| sort @timestamp desc\n| filter @message ~= \"error\"\n| display @timestamp, @message, @logStream\n| limit 100\n",
        "region": "${var.aws_region}",
        "stacked": false,
        "view": "table",
        "title": "${module.update_lease_status_lambda.name} λ error scraper"
      }
    },
    {
        "type": "metric",
        "x": 0,
        "y": 78,
        "width": 12,
        "height": 6,
        "properties": {
            "view": "timeSeries",
            "stacked": false,
            "metrics": [
                [ "AWS/Lambda", "Invocations", "FunctionName", "${module.update_principal_policy.name}", { "color": "#1f77b4" } ],
                [ ".", "Errors", ".", ".", { "color": "#d62728" } ]
            ],
            "region": "${var.aws_region}",
            "title": "${module.update_principal_policy.name} λ"
        }
    },
    {
      "type": "log",
      "x": 12,
      "y": 78,
      "width": 12,
      "height": 6,
      "properties": {
        "query": "SOURCE '/aws/lambda/${module.update_principal_policy.name}' | fields @timestamp, @message\n| sort @timestamp desc\n| filter @message ~= \"error\"\n| display @timestamp, @message, @logStream\n| limit 100\n",
        "region": "${var.aws_region}",
        "stacked": false,
        "view": "table",
        "title": "${module.update_principal_policy.name} λ error scraper"
      }
    }
  ]
}
 EOF
}