#!/usr/bin/env bash
set -euxo pipefail

# Check Go Formatting
test -z "$(go fmt ./...)"

# Run Golint
golangci-lint run --disable-all -E golint ./pkg/...
golangci-lint run --disable-all -E golint ./cmd/...
golangci-lint run --disable-all -E golint ./tests/...

# Check terraform formatting
terraform fmt -check=true ./modules/

# Run tflint
tflint --deep --error-with-issues modules