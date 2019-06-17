#!/bin/bash

# Deploy Redbox application code to AWS Master account
# Requires build artifacts to exist in ./bin/
# Run ./scripts/build.sh to generate artifacts
#
# Usage:
#   ./scripts/deploy.sh <artifact_file> <namespace> <artifact_bucket_name>
#
# Example:
#   ./scripts/deploy.sh ./bin/build_artifacts.zip prod 1234567890-redbox-artifacts-prod

set -euxo pipefail

FILE=$1
namespace=$2
artifactBucket=$3

# check if build_artifacts.zip exists (generated from 'scripts/build.sh')
if [[ -f "$FILE" ]]; then
    # Unzip build_artifacts.zip into the '__artifacts__/' directory
    rm -rf __artifacts__
    unzip ${FILE} -d __artifacts__ 

    # Find all Lambda artifacts and upload them to the S3 artifact bucket
    for i in $(ls -d __artifacts__/lambda/*.zip)
    do
        mod_name=$(basename ${i} | cut -f 1 -d '.')
        fn_name="${mod_name}-${namespace}"
        
        # Upload zip file to S3
        aws s3 cp \
          __artifacts__/lambda/${mod_name}.zip \
          s3://${artifactBucket}/lambda/${mod_name}.zip \
          --sse
        
        # Point Lambda Fn at the new code on S3
        aws lambda update-function-code \
          --function-name ${fn_name} \
          --s3-bucket ${artifactBucket} \
          --s3-key lambda/${mod_name}.zip
        
        # Publish new Function version
        aws lambda publish-version \
          --function-name ${fn_name}
    done
    
    # Upload the Reset CodeBuild Zip to the S3 artifact bucket. CodeBuild should pick this new file up on its next build.
    aws s3 cp \
      __artifacts__/codebuild/reset.zip \
      s3://${artifactBucket}/codebuild/reset.zip \
      --sse

    # Delete the '__artifacts__/' directory after uploading to the s3 artifact bucket 
    rm -rf __artifacts__
else 
    echo "[Error] ${FILE} does not exist yet. Run scripts/build.sh to generate it."
    exit 1
fi
