#!/bin/bash
set -euxo pipefail

# Build all Lambda functions
# Looks for `/cmd/lambda/<name>/main.go`
# and packages into `/bin/lambda/<name>.zip`
for i in $(ls -d cmd/lambda/*)
do
if [ -e $i/main.go ];then
    mod_name=`basename $i`
    cd cmd/lambda/$mod_name
    GOARCH=amd64 GOOS=linux go build -v -o ../../../bin/lambda/$mod_name
    cd ../../..
    zip -j --must-match \
        bin/lambda/$mod_name.zip \
        bin/lambda/$mod_name
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