/*
DEPRECRATED as of v0.21.0

As part of the v0.21.0 release, we are renaming all our DynamoDB tables
to remove the "Redbox" prefix, and to standardize naming conventions.

	RedboxAccountProd 	--> Accounts
	RedboxLeaseProd 		--> Leases
	UsageCache					--> Usage

We are keeping this configuration around for a release cycle,
so we can migrate data off the old tables before destroying
them.
*/

# Redbox Account table
# Tracks the status of AWS Accounts in our pool
resource "aws_dynamodb_table" "redbox_account" {
  name             = "RedboxAccount${title(var.namespace)}"
  read_capacity    = 5
  write_capacity   = 5
  hash_key         = "Id"
  stream_enabled   = true
  stream_view_type = "NEW_AND_OLD_IMAGES"

  global_secondary_index {
    name            = "AccountStatus"
    hash_key        = "AccountStatus"
    projection_type = "ALL"
    read_capacity   = 5
    write_capacity  = 5
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

resource "aws_dynamodb_table" "redbox_lease" {
  name             = "RedboxLease${title(var.namespace)}"
  read_capacity    = 5
  write_capacity   = 5
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
    read_capacity   = 5
    write_capacity  = 5
  }

  global_secondary_index {
    name            = "LeaseStatus"
    hash_key        = "LeaseStatus"
    projection_type = "ALL"
    read_capacity   = 5
    write_capacity  = 5
  }

  global_secondary_index {
    name            = "LeaseId"
    hash_key        = "Id"
    projection_type = "ALL"
    read_capacity   = 5
    write_capacity  = 5
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

resource "aws_dynamodb_table" "usage_cache" {
  name             = "UsageCache${title(var.namespace)}"
  read_capacity    = 5
  write_capacity   = 5
  hash_key         = "StartDate"
  range_key        = "PrincipalId"
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
    name = "StartDate"
    type = "N"
  }

  # TTL enabled attribute
  ttl {
    attribute_name = "TimeToLive"
    enabled        = true
  }

  tags = var.global_tags
}
