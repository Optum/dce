#!/usr/bin/env bash
set -euxo pipefail

mkdir -p junit-report

# Run Unit Tests for 'pkg'
go test -v ./pkg/... 2>&1 | go-junit-report > junit-report/pkg.xml

# Run Unit Tests for 'cmd'
go test -v ./cmd/... 2>&1 | go-junit-report > junit-report/cmd.xml