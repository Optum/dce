# Account table
# Tracks the status of AWS Accounts in our pool
locals {
  // Suffix table names with var.namesapce,
  // unless we're on prod (then no suffix)
  table_suffix = var.namespace == "prod" ? "" : title(var.namespace)
}

resource "aws_dynamodb_table" "accounts" {
  name             = "Accounts${local.table_suffix}"
  read_capacity    = var.accounts_table_rcu
  write_capacity   = var.accounts_table_wcu
  hash_key         = "Id"
  stream_enabled   = true
  stream_view_type = "NEW_AND_OLD_IMAGES"

  global_secondary_index {
    name            = "AccountStatus"
    hash_key        = "AccountStatus"
    projection_type = "ALL"
    read_capacity   = var.accounts_table_rcu
    write_capacity  = var.accounts_table_wcu
  }

  server_side_encryption {
    enabled = true
  }

  # AWS Account ID
  attribute {
    name = "Id"
    type = "S"
  }

  # Status of the Account
  # May be one of:
  #   - LEASED
  #   - READY
  #   - NOT_READY
  attribute {
    name = "AccountStatus"
    type = "S"
  }

  tags = var.global_tags
  /*
  Other attributes:
  - LastModifiedOn (Integer, epoch timestamps)
  - CreatedOn (Integer, epoch timestamps)
  */
}

resource "aws_dynamodb_table" "leases" {
  name             = "Leases${local.table_suffix}"
  read_capacity    = var.leases_table_rcu
  write_capacity   = var.leases_table_wcu
  hash_key         = "AccountId"
  range_key        = "PrincipalId"
  stream_enabled   = true
  stream_view_type = "NEW_AND_OLD_IMAGES"

  server_side_encryption {
    enabled = true
  }

  global_secondary_index {
    name            = "PrincipalId"
    hash_key        = "PrincipalId"
    projection_type = "ALL"
    read_capacity   = var.leases_table_rcu
    write_capacity  = var.leases_table_wcu
  }

  global_secondary_index {
    name            = "LeaseStatus"
    hash_key        = "LeaseStatus"
    projection_type = "ALL"
    read_capacity   = var.leases_table_rcu
    write_capacity  = var.leases_table_wcu
  }

  global_secondary_index {
    name            = "LeaseId"
    hash_key        = "Id"
    projection_type = "ALL"
    read_capacity   = var.leases_table_rcu
    write_capacity  = var.leases_table_wcu
  }

  # AWS Account ID
  attribute {
    name = "AccountId"
    type = "S"
  }

  # Lease status.
  # May be one of:
  # - ACTIVE
  # - INACTIVE
  attribute {
    name = "LeaseStatus"
    type = "S"
  }

  # Principal ID
  attribute {
    name = "PrincipalId"
    type = "S"
  }

  # Lease ID
  attribute {
    name = "Id"
    type = "S"
  }

  tags = var.global_tags
  /*
  Other attributes:
    - LeaseStatusReason (string)
    - CreatedOn (Integer, epoch timestamps)
    - LastModifiedOn (Integer, epoch timestamps)
    - LeaseStatusModifiedOn (Integer, epoch timestamps)
  */
}

resource "aws_dynamodb_table" "usage" {
  name             = "Usage${local.table_suffix}"
  read_capacity    = var.usage_table_rcu
  write_capacity   = var.usage_table_wcu
  hash_key         = "PrincipalId"
  range_key        = "SK"
  stream_enabled   = true
  stream_view_type = "NEW_AND_OLD_IMAGES"

  server_side_encryption {
    enabled = true
  }

  # User Principal ID
  attribute {
    name = "PrincipalId"
    type = "S"
  }

  # AWS usage cost amount for start date as epoch timestamp
  attribute {
    name = "SK"
    type = "S"
  }

  # TTL enabled attribute
  ttl {
    attribute_name = "TimeToLive"
    enabled        = true
  }

  global_secondary_index {
    name            = "SortKey"
    hash_key        = "SK"
    projection_type = "ALL"
    read_capacity   = var.usage_table_rcu
    write_capacity  = var.usage_table_wcu
  }

  tags = var.global_tags
}
