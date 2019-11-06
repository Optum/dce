#!/bin/bash

# Destroy Local DCE deployment
#
# Example:
#   ./scripts/deploy_local/destroy_local_build.sh

set -euxo pipefail

KEY="local-tf-state"
TABLE="local-tf-state"
NAMESPACE=$(whoami)

cd modules
terraform destroy -var="namespace=$NAMESPACE"
rm -rf .terraform
cd ../scripts/deploy_local
terraform destroy -var="namespace=$NAMESPACE"
rm -rf .terraform
