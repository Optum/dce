#!/usr/bin/env bash
#
# testmain.sh
# 

set -o pipefail

workdir=$(dirname $0)
# Do something like this if the test output dir should be unique
# tmpdir=/tmp/dce-tests-$(date +"%y%m%d%H%M")
tmpdir=/tmp/dce-tests
mkdir -p ${tmpdir}
zip_file=${tmpdir}/dce-cli-$(date +"%y%m%d%H%M").zip
dce_config_file=${tmpdir}/dce-cfg.yml
dce_opts="--config ${dce_config_file}"
log_file="${tmpdir}/test.log"

# Zero out the log file before doing anything else...
cat /dev/null > ${log_file}

function writeConfig() {
    cat <<EOF > $1
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

# cleanup ends all leases and removes all accounts from dce master
function cleanup() {
    printTestBanner "Cleanup"

    active_leases=$("${dce_cmd}" leases list 2>&1 | jq -c -r '.[] | select(.leaseStatus=="Active")')
    for lease in $active_leases
    do
        account_id=$(echo $lease | jq -r '.accountId')
        principal_id=$(echo $lease | jq -r '.principalId')
        "${dce_cmd}" leases end --account-id $account_id --principal-id $principal_id >> "${tmpdir}/test.log" 2>&1
        if [[ $? -ne 0 ]]; then
            lease_id=$(echo $lease | jq -r '.id')
            echo "Error while trying to end lease id \"${lease_id}\". Retrying...". 1>&2
            "${dce_cmd}" leases end --account-id $account_id --principal-id $principal_id >> "${tmpdir}/test.log" 2>&1
            if [[ $? -ne 0 ]]; then
                echo "Unable to end lease id \"${lease_id}\"." 1>&2
            fi
        fi
    done

    accounts_ids=$("${dce_cmd}" accounts list 2>&1 | jq -c -r '.[].id')
    for id in $accounts_ids
    do
        "${dce_cmd}" accounts remove $id >> "${tmpdir}/test.log" 2>&1
        if [[ $? -ne 0 ]]; then
            echo "Error while trying to remove account id \"${id}\". Retrying...". 1>&2
            "${dce_cmd}" accounts remove $id >> "${tmpdir}/test.log" 2>&1
            if [[ $? -ne 0 ]]; then
                echo "Unable to remove account id \"${id}\"." 1>&2
            fi
        fi
    done
    printTestResult "Finished"
}

trap cleanup EXIT

# assertExists makes sure the given command exists on the path and is executable.
# Like all of the other assert* functions, this will print a "Failed. ..."
# message and exit
# args:
#     1 - the command name (eg., "whoami", "unzip")
function assertExists() {
    path=$(command -v $1)
    if [[ ! -x "${path}" ]]; then
        echo "Failed. Cannot find command \"${1}\", which is required. Exiting." >&2
        exit -1
    fi
    return 0
}

# assertSuccess makes sure the given command succeeds
# Like all of the other assert* functions, this will print a "Failed. ..."
# message and exit
# args:
#     1 - The tag in the output for the logs. This is used later to find
#         messages in the output with assertInLog or assertNotInLog
#     2.. - The command and all of its arguments
function assertSuccess() {
cat <<EOF >> "${tmpdir}/test.log" 2>&1
Running command "${@:2}":
EOF
    ${@:2} 2>&1 | sed -l "s/^/$1: /" >> "${tmpdir}/test.log"
    if [[ $? -ne 0 ]]; then
        echo "Failed. Error while trying to run \"${@:2}\". check logs." 1>&2
        exit -1
    fi
    printTestResult "Success"
    return 0
}

# assertInLog makes sure the given message is found in the test logs.
# Like all of the other assert* functions, this will print a "Failed. ..."
# message and exit
# args:
#     1 - The tag for the test output to use to filter the results
#     2 - The value to look for in the logs
function assertInLog() {
    echo -n "Asserting message \"$2\" found in logs..."
    echo -n "Asserting message \"$2\" found in logs..." >> "${tmpdir}/test.log" 2>&1
    count=$(grep -e "^$1:" ${log_file} | grep "$2" | wc -l)
    if [[ ${count} -eq 0 ]]; then
        printTestResult "Failed"
        echo "Failed. Expected message \"$2\" for step \"$1\" not found in logs." 1>&2
        exit -1
    elif [[ ${count} -gt 1 ]]; then
        printTestResult "Failed"
        echo "Failed. Expected message \"$2\" for step \"$1\" found more than once." 1>&2
        exit -1
    fi
    printTestResult "Success"
    return 0
}

# assertNotInLog makes sure the given message is NOT found in the test logs.
# Like all of the other assert* functions, this will print a "Failed. ..."
# message and exit
# args:
#     1 - The tag for the test output to use to filter the results
#     2 - The value to look for to be NOT in the logs
function assertNotInLog() {
    echo -n "Asserting message \"$2\" NOT found in logs..."
    echo -n "Asserting message \"$2\" NOT found in logs..." >> "${tmpdir}/test.log" 2>&1
    count=$(grep -e "^$1:" ${log_file} | grep -i "$2" | wc -l)
    if [[ ${count} -ge 1 ]]; then
        printTestResult "Failed"
        echo "Failed. Message \"$2\" for step \"$1\" found but not expected." 1>&2
        exit -1
    fi
    printTestResult "Success"
    return 0
}

# There are a couple things to make sure the user has installed. Like curl and jq (maybe?)
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

writeConfig ${dce_config_file}

#------------------------------------------------------------------------------
# Test 1 - Make sure command works and usage works
#------------------------------------------------------------------------------
# printTestBanner "Get help"
# assertSuccess "help" "${dce_cmd}" --help
# assertInLog "help" "Disposable Cloud Environment"
# assertInLog "help" "Usage:"

# #------------------------------------------------------------------------------
# # Test 2 - Test deployment
# #------------------------------------------------------------------------------
# printTestBanner "Deploy"
# assertSuccess "deploy" "${dce_cmd}" system deploy -b ${dce_opts}
# assertNotInLog "deploy" "fail"
# assertInLog "deploy" "Initializing"
# assertInLog "deploy" "Creating DCE infrastructure"

#------------------------------------------------------------------------------
# Test 2 - Test deployment
#------------------------------------------------------------------------------
function waitForAccountReady() {
    echo "Waiting for account to be ready" | tee -a "${tmpdir}/test.log"
    ready_account=$("${dce_cmd}" accounts describe "${CHILD_ACCOUNT_ID}" 2>&1 | jq -c -r 'select(.accountStatus=="Ready")')
    while [[ -z "${ready_account}" ]]
    do
        sleep 2
        echo "Account is not ready" | tee -a "${tmpdir}/test.log"
        ready_account=$("${dce_cmd}" accounts describe "${CHILD_ACCOUNT_ID}" 2>&1 | jq -c -r 'select(.accountStatus=="Ready")')
    done
    echo "Account is ready" | tee -a "${tmpdir}/test.log"
    return 0
}

function test_deployment() {
    printTestBanner "WHEN an account is leased with 0 lease budget AND the usage state machine wait period is set to 10 seconds THEN the lease should be ended within 15 seconds."
    assertSuccess "adding account" "${dce_cmd}" accounts add --account-id "${CHILD_ACCOUNT_ID}" --admin-role-arn arn:aws:iam::"${CHILD_ACCOUNT_ID}":role/DCEMasterAccess
    assertNotInLog "adding account" "err"
    waitForAccountReady
    assertSuccess "creating lease" "${dce_cmd}" leases create --budget-amount 0 --budget-currency USD --email jane.doe@email.com --principal-id user1
    assertNotInLog "creating lease" "err"
}


test_deployment

exit 0