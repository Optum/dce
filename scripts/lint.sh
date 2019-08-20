#!/usr/bin/env bash
set -euxo pipefail

# Check Go Formatting
gofmtout=$(go fmt ./...)
test -z "${gofmtout}"

# Run Golint
golangci-lint run --disable-all -E golint ./pkg/...
golangci-lint run --disable-all -E golint ./cmd/...
golangci-lint run --disable-all -E golint ./tests/...

# Check terraform formatting
terraform fmt -check=true ./modules/

# Run tflint
cd modules
function moveBackBackend {
  mv ./backend.tf{.bak,} || true
}
trap moveBackBackend EXIT
# Move backend.tf, so we can tf init in a CI environment
mv -f ./backend.tf{,.bak} || true
terraform init
tflint --deep --error-with-issues ./