#!/usr/bin/env bash
set -exuo pipefail

# Create an S3 bucket, to use as a Terraform state backend.
# See https://www.terraform.io/docs/backends/types/s3.html

BUCKET_PREFIX="${1}"

bucket_name="${BUCKET_PREFIX}-dce-tfstate"

# Check if the bucket already exists
set +e
aws s3 ls s3://${bucket_name} > /dev/null 2>&1
listBucketRes=$?
aws sts get-caller-identity
aws configure list-profiles
cat ~/.aws/credentials
if [[ listBucketRes -eq 0 ]]; then
  echo "Using s3://${bucket_name} as Terraform state backend"
else

  set -e

  aws --version
  # Create the S3 bucket
  echo "Creating S3 bucket s3://${bucket_name} to use as terraform state backend... "
  aws s3api create-bucket \
    --bucket ${bucket_name} \
    --acl log-delivery-write > /dev/null
  echo "done."

  # Operations on the bucket seem to fail if immediately after creating
  # the bucket.
  sleep 20

  # Set default encryption on the bucket
  echo "Configuring bucket encryption for s3://${bucket_name}... "
  aws s3api put-bucket-encryption \
    --bucket ${bucket_name} \
    --server-side-encryption-configuration '{"Rules": [{"ApplyServerSideEncryptionByDefault": {"SSEAlgorithm": "AES256"}}]}' \
     > /dev/null
  echo "done."

  # Only allow SSL traffic to bucket
  echo "Configuring bucket access policy for s3://${bucket_name}... "
  aws s3api put-bucket-policy \
    --bucket ${bucket_name} \
    --policy "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Sid\":\"DenyInsecureCommunications\",\"Effect\":\"Deny\",\"Principal\":\"*\",\"Action\":\"s3:*\",\"Resource\":\"arn:aws:s3:::${bucket_name}/*\",\"Condition\":{\"Bool\":{\"aws:SecureTransport\":\"false\"}}}]}" \
     > /dev/null
  echo "done."
fi

# Generate a `backend.tf` file, to point at this backend
echo "Generating backend.tf file... "
scriptDir=$(dirname "${0}")
echo '
terraform {
  backend "s3" {
    bucket = '\""${bucket_name}"\"'
    region = "us-east-1"
    key    = "dce.tfstate"
  }
}
' > ${scriptDir}/../modules/backend.tf
echo "done."

echo "Terraform backend creation complete."
