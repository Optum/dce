#!/bin/bash
set -euxo pipefail

export GOARCH=amd64 GOOS=linux CGO_ENABLED=0


# Accept a Lambda names as the first argument,
# to only build a single lambda
set +u
lambda_target=${1}
if [[ -z ${lambda_target} ]]; then
  lambda_dirs=$(ls -d cmd/lambda/*)
else
  lambda_dirs=$(ls -d cmd/lambda/${lambda_target})
fi
set -u

# Build all Lambda functions
# Looks for `/cmd/lambda/<name>/main.go`
# and packages into `/bin/lambda/<name>.zip`
for i in ${lambda_dirs}
do
if [ -e $i/main.go ];then
    mod_name=`basename $i`
    cd cmd/lambda/$mod_name
    GOARCH=amd64 GOOS=linux go build -v -o ../../../bin/lambda/$mod_name/bootstrap
    cd ../../..
    zip -j --must-match \
      bin/lambda/$mod_name.zip \
      bin/lambda/$mod_name/bootstrap
    # Include static web assets if they exist
    if [ -d "./cmd/lambda/$mod_name/public" ] && [ -d "./cmd/lambda/$mod_name/views" ];then
      cd ./cmd/lambda/$mod_name
      set +e
      zip -u --must-match \
          ../../../bin/lambda/$mod_name.zip \
          public/* \
          views/*
      set -e
      # See https://stackoverflow.com/a/19258421/830030
      zipReturn=$?
      if [[ zipReturn -ne 12 && zipReturn -gt 0 ]]; then
          exit $zipReturn
      fi
      cd ../../..
    fi
fi
done

# Build Account Reset CodeBuild
# Builds to `/bin/codebuild/reset.zip`
cd cmd/codebuild/reset/
GOARCH=amd64 GOOS=linux go build -o ../../../bin/codebuild/reset ./...
cd ../../../
zip -j --must-match \
    bin/codebuild/reset.zip \
    bin/codebuild/reset \
    cmd/codebuild/reset/buildspec.yml \
    cmd/codebuild/reset/default-nuke-config-template.yml

# Build Lambda/CodeBuild Artifact
cd bin
zip --must-match \
    build_artifacts.zip \
    lambda/*.zip \
    codebuild/*.zip

# Build Terraform Artifact
cd ..
zip -r --must-match \
    bin/terraform_artifacts.zip \
    modules/ \
    -x modules/.terraform/\* modules/*.tfstate* modules/*.tfvars modules/*.zip

# Cleanup
rm -rf bin/codebuild bin/lambda
