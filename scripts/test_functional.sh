#!/usr/bin/env bash
set -euxo pipefail

mkdir -p junit-report

# Run functional tests
go test -v ./tests/... -test.timeout 40m 2>&1 | tee >(go-junit-report > junit-report/functional.xml)

