locals {
  dce_principal_policy = var.dce_principal_policy == "" ? "${path.module}/fixtures/policies/dce_principal_policy.tmpl" : var.dce_principal_policy
}


# Configure an S3 Bucket to hold artifacts
# (eg. application code deployments, etc.)
resource "aws_s3_bucket" "artifacts" {
  bucket = "${local.account_id}-dce-artifacts-${var.namespace}"

  # Allow S3 access logs to be written to this bucket
  acl = "log-delivery-write"

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
    target_bucket = "${local.account_id}-dce-artifacts-${var.namespace}"
    target_prefix = "/logs"
  }

  tags = var.global_tags
}

# Enforce SSL only access to the bucket
resource "aws_s3_bucket_policy" "reset_codepipeline_source_ssl_policy" {
  bucket = aws_s3_bucket.artifacts.id

  policy = <<POLICY
{
    "Version": "2012-10-17",
    "Statement": [
      {
        "Sid": "DenyInsecureCommunications",
        "Effect": "Deny",
        "Principal": "*",
        "Action": "s3:*",
        "Resource": "${aws_s3_bucket.artifacts.arn}/*",
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

resource "aws_s3_bucket_object" "dce_principal_policy" {
  bucket = aws_s3_bucket.artifacts.id
  key    = "fixtures/policies/dce_principal_policy.tmpl"
  source = local.dce_principal_policy
  etag   = "${filemd5(local.dce_principal_policy)}"
}
