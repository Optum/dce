#!/usr/bin/env bash
set -euxo pipefail

mkdir -p junit-report

# Run tests
go test -v -coverprofile=coverage.txt -covermode count \
  ./pkg/...  ./cmd/... 2>&1 | \
  tee test.output.txt | \
  tee >(go-junit-report -set-exit-code > junit-report/report.xml)

# Echo Test Output
cat test.output.txt

# Convert coverate to xml and html
gocov convert coverage.txt > coverage.json
gocov-xml < coverage.json > coverage.xml
