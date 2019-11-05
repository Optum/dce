locals {
  principal_policy = var.principal_policy == "" ? "${path.module}/fixtures/policies/principal_policy.tmpl" : var.principal_policy
  artifact_bucket_name = "${local.account_id}-dce-artifacts-${var.namespace}"
}


# Configure an S3 Bucket to hold artifacts
# (eg. application code deployments, etc.)
resource "aws_s3_bucket" "artifacts" {
  bucket = local.artifact_bucket_name

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
    target_bucket = local.artifact_bucket_name
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

resource "aws_s3_bucket_object" "principal_policy" {
  bucket = aws_s3_bucket.artifacts.id
  key    = "fixtures/policies/principal_policy.tmpl"
  source = local.principal_policy
  etag   = "${filemd5(local.principal_policy)}"
}
