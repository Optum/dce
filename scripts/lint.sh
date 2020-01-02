#!/usr/bin/env bash
set -euxo pipefail

# Check Go Formatting
gofmtout=$(go fmt ./...)
test -z "${gofmtout}"

# Run Golint
# TODO: Make sure golangci-lint is installed and ready to be run
GOLANG_LINT_CMD=golangci-lint

if [ ! command -v ${GOLANG_LINT_CMD} ]; then
  echo "$GOLANG_LINT_CMD not found, fetching..."
  go get -u github.com/golangci/golangci-lint/cmd/golangci-lint
fi

golangci-lint run --disable-all -E golint ./pkg/...
golangci-lint run --disable-all -E golint ./cmd/...
golangci-lint run --disable-all -E golint ./tests/...

# Check terraform formatting
terraform fmt -diff -check -recursive ./modules/

# Run tflint
cd modules
function moveBackBackend {
  mv ./backend.tf{.bak,} || true
}
trap moveBackBackend EXIT
# Move backend.tf, so we can tf init in a CI environment
mv -f ./backend.tf{,.bak} || true
terraform init
# TODO: test to see if tflint is installed first.
tflint ./
