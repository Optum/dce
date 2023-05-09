provider "aws" {
  region = "us-east-1"
}

variable "global_tags" {
  description = "The tags to apply to all resources that support tags"
  type        = map(string)

  default = {
    Terraform = "True"
    AppName   = "DCE"
  }
}

variable "namespace" {
  type = string
}

data "aws_caller_identity" "current" {}

# Configure an S3 Bucket to hold artifacts
# (eg. application code deployments, etc.)
resource "aws_s3_bucket" "local_tfstate" {
  bucket = "${data.aws_caller_identity.current.account_id}-local-tfstate-${var.namespace}"

  # Allow S3 access logs to be written to this bucket
  #acl = "log-delivery-write"

  # Allow Terraform to destroy the bucket
  # (so ephemeral PR environments can be torn down)
  force_destroy = true

  # Encrypt objects by default
  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        sse_algorithm = "AES256"
      }
    }
  }

  versioning {
    enabled = true
  }

  # Send S3 access logs for this bucket to itself
  logging {
    target_bucket = "${data.aws_caller_identity.current.account_id}-local-tfstate-${var.namespace}"
    target_prefix = "logs/"
  }

  tags = var.global_tags
}

# Enforce SSL only access to the bucket
resource "aws_s3_bucket_policy" "reset_codepipeline_source_ssl_policy" {
  bucket = aws_s3_bucket.local_tfstate.id

  policy = <<POLICY
{
    "Version": "2012-10-17",
    "Statement": [
      {
        "Sid": "DenyInsecureCommunications",
        "Effect": "Deny",
        "Principal": "*",
        "Action": "s3:*",
        "Resource": "${aws_s3_bucket.local_tfstate.arn}/*",
        "Condition": {
            "Bool": {
                "aws:SecureTransport": "false"
            }
        }
      }
    ]
}
POLICY

}

resource "aws_dynamodb_table" "local_terraform_state_lock" {
  name           = "Terraform-State-Backend-${var.namespace}"
  read_capacity  = 1
  write_capacity = 1
  hash_key       = "LockID"

  attribute {
    name = "LockID"
    type = "S"
  }
}

output "bucket" {
  value = aws_s3_bucket.local_tfstate.bucket
}


