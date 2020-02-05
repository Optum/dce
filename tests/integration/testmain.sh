#!/usr/bin/env bash

workdir=$(dirname $0)
zipfile="/tmp/dce-cli-$(date +"%y%m%d%H%M").zip"

function printTestBanner() {
cat >> "${workdir}/test.log" 2>&1 <<EOF
###############################################################################
Running test: "$1": 
EOF    
    echo -n "Running test: \"${1}\"..."
}

function printTestResult() {
cat >> "${workdir}/test.log" 2>&1 <<EOF
...$1!
###############################################################################
EOF 
    echo "...$1"
}

function assertExists() {
    path=$(command -v $1)
    if [[ ! -x "${path}" ]]; then
        echo "Cannot find command \"${1}\", which is required. Exiting." >&2
        exit -1
    fi
}

function assertExitZ() {
cat >> "${workdir}/test.log" 2>&1 <<EOF
Running command "$@": 
EOF
    "$@" >> "${workdir}/test.log" 2>&1
    if [[ $? -ne 0 ]]; then
        echo "FAIL! Error while trying to run \"$@\". check logs." 1>&2
        exit -1
    fi
    return 0
}

# There are a couple things to make sure the user has installed. Like curl and jq

assertExists "curl"
assertExists "unzip"
# assertExists "jq"

echo -n "Downloading DCE..."

[[ -f ${workdir}/test.cfg ]] && source $workdir/test.cfg

if [[ ${DCE_CLI_VERSION-latest} == "latest" ]]; then 
    echo "Version not specified, so using latest."
    dce_bin_url="https://github.com/Optum/dce-cli/releases/latest/download/dce_darwin_amd64.zip"
else
    echo "Using version ${DCE_CLI_VERSION}."
    dce_bin_url="https://github.com/Optum/dce-cli/releases/download/${DCE_CLI_VERSION}/dce_darwin_amd64.zip"
fi

curl -p# -o ${zipfile} -L ${dce_bin_url}

echo "Finished downloading; extracting file..."

(cd $(dirname "${zipfile}") && unzip -q -o "${zipfile}")

dce_cmd=/tmp/dce

if [[ ! -x "${dce_cmd}" ]]; then
    chomd u+x "${dce_cmd}"
fi

#------------------------------------------------------------------------------
# Test 1 - Make sure command works and usage works
#------------------------------------------------------------------------------
printTestBanner "Get help"
assertExitZ "${dce_cmd}" --help
printTestResult "Success"

exit 0