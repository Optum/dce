resource "aws_cloudwatch_log_group" "account_pool_metrics" {
  name              = "/aws/lambda/account_pool_metrics-${var.namespace}"
  retention_in_days = var.cloudwatch_log_retention
}

import {
  to = aws_cloudwatch_log_group.account_pool_metrics
  id = "/aws/lambda/account_pool_metrics-sandbox-20230905"
}

resource "aws_cloudwatch_log_group" "lease_auth" {
  name              = "/aws/lambda/lease_auth-${var.namespace}"
  retention_in_days = var.cloudwatch_log_retention
}

import {
  to = aws_cloudwatch_log_group.lease_auth
  id = "/aws/lambda/lease_auth-sandbox-20230905"
}

resource "aws_cloudwatch_log_group" "credentials_web_page" {
  name              = "/aws/lambda/credentials_web_page-${var.namespace}"
  retention_in_days = var.cloudwatch_log_retention
}

import {
  to = aws_cloudwatch_log_group.credentials_web_page
  id = "/aws/lambda/credentials_web_page-sandbox-20230905"
}

resource "aws_cloudwatch_log_group" "accounts" {
  name              = "/aws/lambda/accounts-${var.namespace}"
  retention_in_days = var.cloudwatch_log_retention
}

import {
  to = aws_cloudwatch_log_group.accounts
  id = "/aws/lambda/accounts-sandbox-20230905"
}

resource "aws_cloudwatch_log_group" "leases" {
  name              = "/aws/lambda/leases-${var.namespace}"
  retention_in_days = var.cloudwatch_log_retention
}

import {
  to = aws_cloudwatch_log_group.leases
  id = "/aws/lambda/leases-sandbox-20230905"
}

resource "aws_cloudwatch_log_group" "populate_reset_queue" {
  name              = "/aws/lambda/populate_reset_queue-${var.namespace}"
  retention_in_days = var.cloudwatch_log_retention
}

import {
  to = aws_cloudwatch_log_group.populate_reset_queue
  id = "/aws/lambda/populate_reset_queue-sandbox-20230905"
}

resource "aws_cloudwatch_log_group" "process_reset_queue" {
  name              = "/aws/lambda/process_reset_queue-${var.namespace}"
  retention_in_days = var.cloudwatch_log_retention
}

import {
  to = aws_cloudwatch_log_group.process_reset_queue
  id = "/aws/lambda/process_reset_queue-sandbox-20230905"
}

resource "aws_cloudwatch_log_group" "fan_out_update_lease_status" {
  name              = "/aws/lambda/fan_out_update_lease_status-${var.namespace}"
  retention_in_days = var.cloudwatch_log_retention
}

import {
  to = aws_cloudwatch_log_group.fan_out_update_lease_status
  id = "/aws/lambda/fan_out_update_lease_status-sandbox-20230905"
}

resource "aws_cloudwatch_log_group" "update_lease_status" {
  name              = "/aws/lambda/update_lease_status-${var.namespace}"
  retention_in_days = var.cloudwatch_log_retention
}

import {
  to = aws_cloudwatch_log_group.update_lease_status
  id = "/aws/lambda/update_lease_status-sandbox-20230905"
}

resource "aws_cloudwatch_log_group" "update_principal_policy" {
  name              = "/aws/lambda/update_principal_policy-${var.namespace}"
  retention_in_days = var.cloudwatch_log_retention
}

import {
  to = aws_cloudwatch_log_group.update_principal_policy
  id = "/aws/lambda/update_principal_policy-sandbox-20230905"
}

resource "aws_cloudwatch_log_group" "usage" {
  name              = "/aws/lambda/usage-${var.namespace}"
  retention_in_days = var.cloudwatch_log_retention
}

# import {
#   to = aws_cloudwatch_log_group.usage
#   id = "/aws/lambda/usage-sandbox-20230905"
# }