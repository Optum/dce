resource "aws_s3_bucket" "aws_nuke_config" {
  bucket = var.reset_nuke_template_bucket

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

  tags = var.global_tags
}

resource "aws_s3_bucket_object" "aws_nuke_config" {
  bucket = local.aws_nuke_config_bucket
  key    = var.reset_nuke_template_key
  source = "${path.module}/${var.reset_nuke_template_key}"
}