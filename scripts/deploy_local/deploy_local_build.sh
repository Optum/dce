#!/bin/bash

# Deploy Local Redbox application deployment
# 
# Example:
#   ./scripts/deploy_local/deploy_local_build.sh

set -euxo pipefail

KEY="local-tf-state"
REGION="us-east-1"
TABLE="local-tf-state"
NAMESPACE=$(whoami)

cd scripts/deploy_local && terraform init
terraform apply -var="namespace=$NAMESPACE"
BUCKET=$(terraform output bucket)
cd ../../modules
terraform init -backend-config="bucket=$BUCKET" -backend-config="key=$KEY"
terraform apply -var="namespace=$NAMESPACE"
ARTBUCKET=$(terraform output artifacts_bucket_name)
cd ../
scripts/deploy.sh $NAMESPACE $ARTBUCKET