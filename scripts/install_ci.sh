#!/bin/bash
set -euxo pipefail

export GOBIN=$(dirname `which go`)

curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $GOBIN v1.17.1

wget -q https://github.com/wata727/tflint/releases/download/v0.10.1/tflint_linux_amd64.zip
unzip tflint_linux_amd64.zip
chmod +x tflint
mv tflint $GOBIN

go get github.com/jstemmer/go-junit-report
go get github.com/axw/gocov/gocov
go get github.com/AlekSi/gocov-xml
go get github.com/matm/gocov-html
