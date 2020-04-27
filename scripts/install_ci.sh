#!/bin/bash
#------------------------------------------------------------------------------
# install_ci.sh
# The purpose of this script is install the required tools for building and
# testing DCE
#------------------------------------------------------------------------------

set -euxo pipefail

# safeGoGet will get the go tool, checking to see if it exists first and doing
# a little additional messaging
function safeGoGet() {
  binname=$(basename $1)
  echo "Checking for ${binname} on the path..."
  if [ ! -x "$(command -v ${binname})" ]; then 
    echo -n "Getting ${binname} from $1..."
    go get $1
    echo "Done."
  fi
}

if [ ! -x "$(command -v go)" ]; then
  echo "Cannot find \'go\'. Check your $PATH to make sure it includes \'go\'." >&2
  exit 1
fi

gopath=$(command -v go)
export GOBIN=$(dirname ${gopath})
# Must be enabled to install certain tools
# eg. https://github.com/golangci/golangci-lint/issues/1040#issuecomment-618269286
export GO111MODULE=on

# Get the proper version of tflint first and install it into the same path
# as the other go tools.
if [ -x "$(command -v wget)" ]; then
  if [ ! -x "$(command -v tflint)" ]; then
    if [[ $(uname -s) == "Darwin" ]]; then
      wget -q https://github.com/wata727/tflint/releases/download/v0.15.4/tflint_darwin_amd64.zip -O /tmp/tflint.zip
    else
      wget -q https://github.com/wata727/tflint/releases/download/v0.15.4/tflint_linux_amd64.zip -O /tmp/tflint.zip
    fi
    (cd /tmp && unzip tflint.zip)
    chmod +x /tmp/tflint
    mv /tmp/tflint $GOBIN
  fi
else
  echo "Cannot find wget, which is required for installing tools."
  exit 1
fi

#------------------------------------------------------------------------------
# Dependencies go here.
#------------------------------------------------------------------------------

# go-junit-report is used by the test scripts to generate report output readable
# by CI tools that can read JUnit reports
safeGoGet github.com/jstemmer/go-junit-report

# gcov is used to generate coverage information in a report that can be read
# by CI tools
safeGoGet github.com/axw/gocov/gocov
safeGoGet github.com/AlekSi/gocov-xml
safeGoGet github.com/matm/gocov-html

# golangci-lint is a lint aggregator used in the lint.sh script to lint the
# go and terraform code.
safeGoGet github.com/golangci/golangci-lint/cmd/golangci-lint

# gosec is used for checking the code for security problems
safeGoGet github.com/securego/gosec/cmd/gosec

echo "Setup complete."
