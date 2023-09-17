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

set -euxo pipefail

FILE="$1"
NAMESPACE="$2"
ARTIFACT_BUCKET="$3"
AWS_PROFILE="$4"

# Check if build_artifacts.zip exists (generated from 'scripts/build.sh')
if [[ -f "$FILE" ]]; then
    # Unzip build_artifacts.zip into the '__artifacts__/' directory
    rm -rf __artifacts__
    unzip "$FILE" -d __artifacts__ 

    # Find all Lambda artifacts and upload them to the S3 artifact bucket
    for i in $(ls -d __artifacts__/lambda/*.zip)
    do
        MOD_NAME=$(basename ${i} | cut -f 1 -d '.')
        FN_NAME="${MOD_NAME}-${NAMESPACE}"
        
        # # Upload zip file to S3
        # aws s3 cp \
        #   "__artifacts__/lambda/${MOD_NAME}.zip" \
        #   "s3://${ARTIFACT_BUCKET}/lambda/${MOD_NAME}.zip" \
        #   --sse

        aws s3 cp \
          "__artifacts__/lambda/${MOD_NAME}.zip" \
          "s3://${ARTIFACT_BUCKET}/lambda/${MOD_NAME}.zip" \
          --sse \
          --profile $AWS_PROFILE
        

        # Point Lambda Fn at the new code on S3 and publish new version
        # aws lambda update-function-code \
        #   --function-name "${FN_NAME}" \
        #   --s3-bucket "${ARTIFACT_BUCKET}" \
        #   --s3-key "lambda/${MOD_NAME}.zip" \
        #   --publish

        aws lambda update-function-code \
          --function-name "${FN_NAME}" \
          --s3-bucket "${ARTIFACT_BUCKET}" \
          --s3-key "lambda/${MOD_NAME}.zip" \
          --publish \
          --profile $AWS_PROFILE
    done
    
    # Upload the Reset CodeBuild Zip to the S3 artifact bucket. CodeBuild should pick this new file up on its next build.
    # aws s3 cp \
    #   __artifacts__/codebuild/reset.zip \
    #  "s3://${ARTIFACT_BUCKET}/codebuild/reset.zip" \
    #   --sse

    aws s3 cp \
      __artifacts__/codebuild/reset.zip \
     "s3://${ARTIFACT_BUCKET}/codebuild/reset.zip" \
      --sse \
      --profile $AWS_PROFILE

    # Delete the '__artifacts__/' directory after uploading to the s3 artifact bucket 
    rm -rf __artifacts__
else 
    echo "[Error] ${FILE} does not exist yet. Run scripts/build.sh to generate it."
    exit 1
fi
