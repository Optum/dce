#!/usr/bin/env bash
set -euo pipefail

echo -n "Formatting golang code... "
gofmtout=$(go fmt ./...)
if [ "$gofmtout" ]; then
  printf "\n\n"
  echo "Files with formatting errors:"
  echo "${gofmtout}"
  exit 1
fi
echo "done."

echo -n "Linting golang code... "
# TODO: Make sure golangci-lint is installed and ready to be run
GOLANG_LINT_CMD=golangci-lint

if [ ! "$(command -v ${GOLANG_LINT_CMD})" ]; then
  echo -n "installing ${GOLANG_LINT_CMD}... "
  go get -u github.com/golangci/golangci-lint/cmd/golangci-lint
fi

golangci-lint run \
  --exclude "composite literal uses unkeyed fields" \
  --exclude '`main` is unused' \
  --timeout 5m0s
echo "done."

echo -n "Formatting terraform code.... "
terraform fmt -diff -check -recursive ./modules/
echo "done."

# Run tflint
echo -n "Linting terraform code... "
cd modules
terraform init &> /dev/null
# TODO: test to see if tflint is installed first.
tflint --deep ./
echo -e '\b done.'
