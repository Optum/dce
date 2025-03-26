#!/bin/bash

set -euxo pipefail

FILE="$1"
NAMESPACE="$2"
ARTIFACT_BUCKET="$3"

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
        
        # Upload zip file to S3
        aws s3 cp \
          "__artifacts__/lambda/${MOD_NAME}.zip" \
          "s3://${ARTIFACT_BUCKET}/lambda/${MOD_NAME}.zip" \
          --sse || {
            echo "[Error] Failed to upload __artifacts__/lambda/${MOD_NAME}.zip to s3://${ARTIFACT_BUCKET}/lambda/${MOD_NAME}.zip"
            exit 1
        }
        
        # Point Lambda Fn at the new code on S3 and publish new version
        aws lambda update-function-code \
          --function-name "${FN_NAME}" \
          --s3-bucket "${ARTIFACT_BUCKET}" \
          --s3-key "lambda/${MOD_NAME}.zip" || {
            echo "[Error] Failed to update Lambda function ${FN_NAME} with new code from s3://${ARTIFACT_BUCKET}/lambda/${MOD_NAME}.zip"
            exit 1
        }
    done

    # Clean up the '__artifacts__/' directory
    rm -rf __artifacts__
else
    echo "[Error] $FILE does not exist yet. Run scripts/build.sh to generate it."
    exit 1
fi