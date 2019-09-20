#!/usr/bin/env bash
set -euxo pipefail

mkdir -p junit-report

# Run functional tests
go test -v ./tests/acceptance/usage_test.go 2>&1 | go-junit-report > junit-report/functional.xml

