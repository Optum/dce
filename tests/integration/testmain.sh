#!/usr/bin/env bash

workdir=$(dirname $0)
# tmpdir="/tmp/dce-tests-$(date +"%y%m%d%H%M")"
tmpdir="/tmp/dce-tests"
mkdir -p ${tmpdir}
zip_file="${tmpdir}/dce-cli-$(date +"%y%m%d%H%M").zip"
dce_config_file="${tmpdir}/dce-cfg.yml"
dce_opts="--config ${dce_config_file}"

function writeConfig() {
    echo <<EOF > $1
api:
  host:
  basepath:
region: us-east-1
EOF

}

function printTestBanner() {
cat <<EOF >> "${tmpdir}/test.log" 2>&1
###############################################################################
Running test: "$1" 
EOF
    echo -n "Running test: \"${1}\"..."
    return 0
}

function printTestResult() {
cat <<EOF >> "${tmpdir}/test.log" 2>&1
...$1!
-------------------------------------------------------------------------------
EOF
    echo "...$1!"
    return 0
}

function assertExists() {
    path=$(command -v $1)
    if [[ ! -x "${path}" ]]; then
        echo "Cannot find command \"${1}\", which is required. Exiting." >&2
        exit -1
    fi
    return 0
}

function assertSuccess() {
cat <<EOF >> "${tmpdir}/test.log" 2>&1
Running command "$@": 
EOF
    $@ >> "${tmpdir}/test.log" 2>&1
    if [[ $? -ne 0 ]]; then
        echo "Failed. Error while trying to run \"$@\". check logs." 1>&2
        exit -1
    fi
    printTestResult "Success"  
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

curl -p# -o ${zip_file} -L ${dce_bin_url}

echo "Finished downloading; extracting file..."

(cd $(dirname "${zip_file}") && unzip -q -o "${zip_file}")

dce_cmd=$(dirname "${zip_file}")/dce

if [[ ! -x "${dce_cmd}" ]]; then
    chmod u+x "${dce_cmd}"
fi
rm -f "${zip_file}"

writeConfig "${dce_config_file}"

#------------------------------------------------------------------------------
# Test 1 - Make sure command works and usage works
#------------------------------------------------------------------------------
printTestBanner "Get help"
assertSuccess "${dce_cmd}" --help

#------------------------------------------------------------------------------
# Test 1 - Test deployment
#------------------------------------------------------------------------------
printTestBanner "Deploy"
assertSuccess "${dce_cmd}" system deploy -b ${dce_opts}

exit 0
