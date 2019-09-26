
#!/bin/bash

# Destroy Local Redbox application deployment
#
# Example:
#   ./scripts/deploy_local/destroy_local_build.sh

set -uexo pipefail

KEY="local-tf-state"
TABLE="local-tf-state"
NAMESPACE=$(whoami)

cd modules
terraform destroy -var="namespace=$NAMESPACE"
rm -rf .terraform
cd ../scripts/deploy_local
terraform destroy -var="namespace=$NAMESPACE"
rm -rf .terraform