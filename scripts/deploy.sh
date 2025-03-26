#!/bin/bash

# Deploy DCE to AWS Master account
# Requires build artifacts to exist in ./bin/
# Run ./scripts/build.sh to generate artifacts
#
# Usage:
#   ./scripts/deploy.sh <artifact_file> <namespace> <artifact_bucket_name>
#
# Example:
#   ./scripts/deploy.sh ./bin/build_artifacts.zip prod 1234567890-dce-artifacts-prod


# Ensure the script exits on any error
set -e

# Define the artifact bucket
ARTIFACT_BUCKET=$3

# Find all Lambda artifacts and upload them to the S3 artifact bucket
# Check if the lambda zip files exist
if ls __artifacts__/lambda/*.zip 1> /dev/null 2>&1; then
  for file in __artifacts__/lambda/*.zip; do
    MOD_NAME=$(basename "$file" .zip)
    FN_NAME="${MOD_NAME}-${2}"  

    # Upload zip file to S3
    aws s3 cp \
      "__artifacts__/lambda/${MOD_NAME}.zip" \
      "s3://${ARTIFACT_BUCKET}/lambda/${MOD_NAME}.zip" \
      --sse
    
    # Point Lambda Fn at the new code on S3 and publish new version
    aws lambda update-function-code \
      --function-name "${FN_NAME}" \
      --s3-bucket "${ARTIFACT_BUCKET}" \
      --s3-key "lambda/${MOD_NAME}.zip" \
      --publish
  done
else
  echo "[Error] No lambda zip files found in __artifacts__/lambda/"
  exit 1
fi

# Check if the CodeBuild reset zip file exists
if [ -f "__artifacts__/codebuild/reset.zip" ]; then
  # Upload the Reset CodeBuild Zip to the S3 artifact bucket. CodeBuild should pick this new file up on its next build.
  aws s3 cp \
    __artifacts__/codebuild/reset.zip \
    "s3://${ARTIFACT_BUCKET}/codebuild/reset.zip" \
    --sse

  # Delete the '__artifacts__/' directory after uploading to the s3 artifact bucket 
  rm -rf __artifacts__
else
  echo "[Error] __artifacts__/codebuild/reset.zip does not exist."
  exit 1
fi




