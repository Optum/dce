#!/bin/bash
set -euxo pipefail

export GOBIN=$(dirname `which go`)

export GO111MODULE=off

wget -q https://github.com/wata727/tflint/releases/download/v0.13.4/tflint_linux_amd64.zip
unzip tflint_linux_amd64.zip
chmod +x tflint
mv tflint $GOBIN

go get github.com/jstemmer/go-junit-report
go get github.com/axw/gocov/gocov
go get github.com/AlekSi/gocov-xml
go get github.com/matm/gocov-html
go get -u github.com/golangci/golangci-lint/cmd/golangci-lint

go get github.com/securego/gosec/cmd/gosec
