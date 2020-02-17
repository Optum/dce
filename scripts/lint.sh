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

golangci-lint run
echo "done."

GOSEC_VERSION=v2.2.0
curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s -- -b . $GOSEC_VERSION
./gosec ./...
rm gosec

echo -n "Formatting terraform code.... "
terraform fmt -diff -check -recursive ./modules/
echo "done."

# Run tflint
echo -n "Linting terraform code... "
cd modules
terraform init &> /dev/null
# TODO: test to see if tflint is installed first.
tflint --deep ./ | (grep -v "Awesome" || true)
echo -e '\b done.'
