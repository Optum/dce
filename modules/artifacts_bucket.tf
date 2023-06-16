locals {
  principal_policy     = var.principal_policy == "" ? "${path.module}/fixtures/policies/principal_policy.tmpl" : var.principal_policy
  artifact_bucket_name = "${local.account_id}-dce-artifacts-${var.namespace}"
}


# Configure an S3 Bucket to hold artifacts
# (eg. application code deployments, etc.)
resource "aws_s3_bucket" "artifacts" {
  bucket = local.artifact_bucket_name

  # Allow S3 access logs to be written to this bucket
  # acl = "log-delivery-write"

  # Allow Terraform to destroy the bucket
  # (so ephemeral PR environments can be torn down)
  force_destroy = true

  tags = var.global_tags
}

# Encrypt objects by default
resource "aws_s3_bucket_server_side_encryption_configuration" "artifacts" {
  bucket = aws_s3_bucket.artifacts.bucket

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_versioning" "artifacts" {
  bucket = aws_s3_bucket.artifacts.id
  versioning_configuration {
    status = "Enabled"
  }
}

# Allow S3 access logs to be written to this bucket
resource "aws_s3_bucket_acl" "artifacts" {
  bucket = aws_s3_bucket.artifacts.id
  acl    = "log-delivery-write"
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

resource "aws_s3_object" "principal_policy" {
  bucket = aws_s3_bucket.artifacts.id
  key    = "fixtures/policies/principal_policy.tmpl"
  source = local.principal_policy
  etag   = filemd5(local.principal_policy)
}
