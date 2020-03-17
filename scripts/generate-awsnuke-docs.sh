#!/usr/bin/env bash
set -euo pipefail

#------------------------------------------------------------------------------
# generate-awsnuke-docs.sh
# The purpose of this script is to run the internal tool that generates the
# Markdown of the latest supported services from AWS Nuke. See the 
# tools/awsnukedocgen/README.md file for more details about how the tool works
# and how to tweak it, if necessary.
#------------------------------------------------------------------------------

scriptname=$(basename $0)
# This is the Optum branch of AWS Nuke.
awsnukegit="https://github.com/Optum/aws-nuke.git"
basedir=$(dirname $0)
workdir=$(cd $basedir && cd .. && pwd -P)
builddir=${workdir}/build 
tooldir=${workdir}/tools/awsnukedocgen
toolcmd="go run main.go"

# Check to see if a couple things are installed...

# assertExists makes sure the given command exists on the path and is executable.
# Like all of the other assert* functions, this will print a "Failed. ..."
# message and exit
# args:
#     1 - the command name (eg., "whoami", "unzip")
function assertExists() {
    path=$(command -v $1)
    if [[ ! -x "${path}" ]]; then
        echo "${scriptname}: Failed. Cannot find command \"${1}\", which is required. Exiting." >&2
        exit -1
    fi
    return 0
}

assertExists git
assertExists curl

echo -n "Building documentation for AWS Nuke support... "

mkdir -p ${builddir} > /dev/null 2>&1

curl -s https://awspolicygen.s3.amazonaws.com/js/policies.js \
    -o ${builddir}/policies.js

if [[ $? -ne 0 ]]; then
    echo "${scriptname}: Failed. Error while trying to get policies.js." 1>&2
    exit -1
fi

cat ${builddir}/policies.js | sed 's/app.PolicyEditorConfig=//' > ${builddir}/policies.json

if [[ $? -ne 0 ]]; then
    echo "${scriptname}: Failed. Error while trying to create policies.json." 1>&2
    exit -1
fi

rm -rf ${builddir}/aws-nuke
(cd ${builddir} && git clone ${awsnukegit} aws-nuke > /dev/null 2>&1)

if [[ $? -ne 0 ]]; then
    echo "${scriptname}: Failed. Error while trying to clone aws-nuke locally." 1>&2
    exit -1
fi

# These flags will use the tool to generate the markdown file that
# contains 
cd ${tooldir} 
${toolcmd} -nuke-source-dir=${builddir}/aws-nuke -policies-js-file=${builddir}/policies.json -generate-markdown > ${builddir}/awsnuke-support.md 2>/dev/null

if [[ $? -ne 0 ]]; then
    echo "${scriptname}: Failed. Error while trying to clone aws-nuke locally." 1>&2
    exit -1
fi

mv -f ${builddir}/awsnuke-support.md ${workdir}/docs

echo -e '\b done.'
