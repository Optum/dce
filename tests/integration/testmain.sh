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

function endLease() {
  leaseId="${1}"
  accountId=$("${dce_cmd}" leases describe "${leaseId}" 2>&1 | jq -r '.accountId')
  principalId=$("${dce_cmd}" leases describe "${leaseId}" 2>&1 | jq -r '.principalId')
  "${dce_cmd}" leases end --account-id "${accountId}" --principal-id "${principalId}" | tee -a "${tmpdir}/test.log"
  err=$?
  if ((err == 0)); then
      echo "Successfully ended lease ${leaseId}" >> "${tmpdir}/test.log"
  else
    echo "${err}" | tee -a "${tmpdir}/test.log"
  fi
}

function endAllLeases() {
    active_leases=$("${dce_cmd}" leases list --status "Active" 2>&1 | jq -c '.[]')
    for lease in ${active_leases}
    do
      endLease "$(echo "${lease}" | jq -r '.id')"
    done
}

function removeAccount() {
  accountId="${1}"
  "${dce_cmd}" accounts remove "${accountId}" | tee -a "${tmpdir}/test.log"
  err=$?
  if ((err == 0)); then
      echo "Successfully removed account ${accountId}" >> "${tmpdir}/test.log"
  else
    echo "${err}" | tee -a "${tmpdir}/test.log"
  fi
}

function removeAllAccounts() {
    accountsIds=$("${dce_cmd}" accounts list 2>&1 | jq -c -r '.[].id')
    for accountId in ${accountsIds}
    do
      removeAccount "${accountId}"
    done
}

function cleanup() {
    printTestBanner "Cleanup"
    endAllLeases
    removeAllAccounts
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

# waitForAccountStatus waits until the account is in the given status
function waitForAccountStatus() {
    accountId="${1}"
    status="${2}"
    maxWait="${3}"
    echo "Waiting for account to be ${status}" | tee -a "${tmpdir}/test.log"

    matching_account=$("${dce_cmd}" accounts describe "${accountId}" 2>&1 | jq -c -r 'select(.accountStatus=="'${status}'")')
    t=0
    while [[ -z "${matching_account}" ]]
    do
        if ((t >= maxWait)); then
            echo "Ran out of time while waiting for account to become '${status}'" | tee -a "${tmpdir}/test.log"
            exit 1
        fi
        sleep 2
        echo "Account is not '${status}'" | tee -a "${tmpdir}/test.log"
        matching_account=$("${dce_cmd}" accounts describe "${accountId}" 2>&1 | jq -c -r 'select(.accountStatus=="'${status}'")')
        ((t=t+2))
    done

    echo "Account is '${status}'" | tee -a "${tmpdir}/test.log"
    echo "${matching_account}" >> "${tmpdir}/test.log"
    return 0
}

# waitForLeaseStatus waits until the account is ready
function waitForLeaseStatus() {
    leaseId="${1}"
    status="${2}"
    maxWait="${3}"
    echo "Waiting for lease to be '${status}'" | tee -a "${tmpdir}/test.log"

    matching_leases=$("${dce_cmd}" leases describe "${1}" 2>&1 | jq -c -r 'select(.leaseStatus=="'${status}'")')
    t=0
    while [[ -z "${matching_leases}" ]]
    do
        if ((t >= maxWait)); then
            echo "Ran out of time while waiting for lease to become '${status}'" | tee -a "${tmpdir}/test.log"
            exit 1
        fi
        sleep 2
        echo "Lease is not '${status}'" | tee -a "${tmpdir}/test.log"
        matching_leases=$("${dce_cmd}" leases describe "${leaseId}" 2>&1 | jq -c -r 'select(.leaseStatus=="'${status}'")')
        ((t=t+2))
    done

    echo "Lease is '${status}'" | tee -a "${tmpdir}/test.log"
    echo "${matching_leases}" >> "${tmpdir}/test.log"
    return 0
}

# waitForStateMachine sleeps for the wait period of the state machine
function waitForStateMachine() {
    echo "Waiting for state machine to be ready" | tee -a "${tmpdir}/test.log"
    sleepTime=$(aws stepfunctions describe-state-machine --state-machine-arn arn:aws:states:us-east-1:"${MASTER_ACCOUNT_ID}":stateMachine:lease-usage-usg1 | jq -r '.definition' | jq '.States.Wait.Seconds')
    ((sleepTime=sleepTime+5))
    t=1
    until (( t >= sleepTime ))
    do
        echo "Waiting for state machine "${t}"/""${sleepTime}"" seconds."
        sleep 1
        ((t++))
    done
}

# createLease creates a lease and echos the id
function createLease() {
    principalId="${1}"
    budget="${2}"
    leaseDescription=$("${dce_cmd}" leases create --budget-amount "${budget}" --budget-currency USD --email jane.doe@email.com --principal-id "${principalId}" 2>&1)
    leaseId=$(echo "${leaseDescription//Lease created: /}" | jq -r '.id')
    echo "${leaseId}"
    return 0
}

#generateRandString generates a random alphanumeric string
function generateRandString() {
  length="${1}"
  openssl rand -base64 "${length}" | tr -dc A-Za-z0-9
}

function title() {
  toCapitalize="${1}"
  echo "$(tr '[:lower:]' '[:upper:]' <<< ${toCapitalize:0:1})${toCapitalize:1}"
}

# getUsageDailyRecord gets a single usage record for the principalID and LeaseID
function getUsageDailyRecord() {
  timestampPrefix=$(x=$(date +%s); echo ${x:0:3})
  aws dynamodb query --table-name "Principal$(title ${DEPLOY_NAMESPACE})" --key-condition-expression "PrincipalId = :name and begins_with(SK, :sk)" --expression-attribute-values  '{":name":{"S":"'${1}'"}, ":sk":{"S":"Usage-Lease-Daily-'${2}'-'${timestampPrefix}'"}}' | jq -r '.Items[0]'
}

function simulateUsage() {
    principalId="${1}"
    leaseId="${2}"
    usageAmount="${3}"
    echo "Simulating new usage by setting daily usage record CostAmount to -'${usageAmount}'" | tee -a "${tmpdir}/test.log"
    usageRecord=$(getUsageDailyRecord "${principalId}" "${leaseId}")
    subtractedCostAmountRecord=$( echo "${usageRecord}" | jq 'setpath(["CostAmount", "N"]; "-'${usageAmount}'")')
    aws dynamodb put-item --table-name "Principal$(title ${DEPLOY_NAMESPACE})" --item "${subtractedCostAmountRecord}"
}

function getLeaseUsageSummaryRecord() {
    principalId="${1}"
    leaseId="${2}"
    aws dynamodb query \
        --table-name "Principal$(title ${DEPLOY_NAMESPACE})" \
        --key-condition-expression "PrincipalId = :name and begins_with(SK, :sk)" \
        --expression-attribute-values  '{":name":{"S":"'${principalId}'"}, ":sk":{"S":"Usage-Lease-Summary-'${leaseId}'"}}' | jq -r '.Items[0]'
}

# There are a couple things to make sure the user has installed. Like curl and jq (maybe?)
assertExists "curl"
assertExists "unzip"
assertExists "jq"

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
function dce_root_command_works() {
    printTestBanner "Get help"
    assertSuccess "help" "${dce_cmd}" --help
    assertInLog "help" "Disposable Cloud Environment"
    assertInLog "help" "Usage:"
}
# #------------------------------------------------------------------------------
# # Test 2 - Test deployment
# #------------------------------------------------------------------------------
function dce_deploy() {
    printTestBanner "Deploy"
    assertSuccess "deploy" "${dce_cmd}" system deploy -b ${dce_opts}
    assertNotInLog "deploy" "fail"
    assertInLog "deploy" "Initializing"
    assertInLog "deploy" "Creating DCE infrastructure"
}

#------------------------------------------------------------------------------
# Test 2 - Test Lease should end when over lease budget
#------------------------------------------------------------------------------

function test_lease_should_end_when_over_lease_budget() {
    printTestBanner "WHEN a lease goes over the lease budget THEN it should be ended."

    assertSuccess "adding account" "${dce_cmd}" accounts add \
        --account-id "${CHILD_ACCOUNT_ID}" \
        --admin-role-arn arn:aws:iam::"${CHILD_ACCOUNT_ID}":role/DCEMasterAccess
    assertNotInLog "adding account" "err"
    waitForAccountStatus "${CHILD_ACCOUNT_ID}" "Ready" 60

    principalId=$(generateRandString 10)
    echo "principalId is ${principalId}" | tee -a "${tmpdir}/test.log"
    leaseId=$(createLease "${principalId}" 1)
    echo "leaseId is ${leaseId}" | tee -a "${tmpdir}/test.log"
    assertNotInLog "creating lease" "err"

    waitForStateMachine
    simulateUsage "${principalId}" "${leaseId}" 10

    waitForStateMachine
    waitForLeaseStatus "${leaseId}" "Inactive" 5
}

#------------------------------------------------------------------------------
# Test 3 - Test Lease should end when over principal budget
#------------------------------------------------------------------------------

function test_lease_should_end_when_over_principal_budget() {
    printTestBanner "WHEN a lease goes over the principal budget THEN it should be ended."

    assertSuccess "adding account" "${dce_cmd}" accounts add --account-id "${CHILD_ACCOUNT_ID}" --admin-role-arn arn:aws:iam::"${CHILD_ACCOUNT_ID}":role/DCEMasterAccess
    assertNotInLog "adding account" "err"
    waitForAccountStatus "${CHILD_ACCOUNT_ID}" "Ready" 60

    principalId=$(generateRandString 10)
    echo "principalId is ${principalId}" | tee -a "${tmpdir}/test.log"
    principalBudget=$(aws lambda get-function --function-name end_over_budget_lease-"${DEPLOY_NAMESPACE}" --output text --query 'Configuration.Environment.Variables.PRINCIPAL_BUDGET_AMOUNT')
    echo principalBudget is $principalBudget
    ((leaseBudget=principalBudget))
    echo "setting leaseBudget to $leaseBudget"
    leaseId=$(createLease "${principalId}" "${leaseBudget}")
    echo "leaseId is ${leaseId}" | tee -a "${tmpdir}/test.log"
    assertNotInLog "creating lease" "err"
    waitForStateMachine

    # Simulating usage so lease summary record is created
    simulateUsage "${principalId}" "${leaseId}" 0.1
    waitForStateMachine

    # Reduce lease usage summary so that 'over lease budget' is not triggered
    ((usageIncrement=principalBudget+1))
    leaseUsageSummaryRecord=$(getLeaseUsageSummaryRecord "${principalId}" "${leaseId}")
    subtractedCostLeaseSummaryRecord=$( echo "${leaseUsageSummaryRecord}" | jq 'setpath(["CostAmount", "N"]; "-'${usageIncrement}'")')
    aws dynamodb put-item --table-name "Principal$(title ${DEPLOY_NAMESPACE})" --item "${subtractedCostLeaseSummaryRecord}" | tee -a "${tmpdir}/test.log"

    # Simulating usage above principal budget
    simulateUsage "${principalId}" "${leaseId}" "${usageIncrement}"
    waitForStateMachine

    waitForLeaseStatus "${leaseId}" "Inactive" 5
}

# TODO: run select test cases from args OR run all
dce_root_command_works
#dce_deploy
test_lease_should_end_when_over_lease_budget
cleanup
test_lease_should_end_when_over_principal_budget

exit 0