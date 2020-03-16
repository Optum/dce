#!/usr/bin/env bash
# set -euo pipefail

awsnukegit="https://github.com/Optum/aws-nuke.git"
basedir=$(dirname $0)
workdir=$(cd $basedir && cd .. && pwd -P)
builddir=${workdir}/build 
tooldir=${workdir}/tools/iampolgen
toolcmd="go run main.go"

# Check to see if a couple things are installed...

echo -n "Building documentation for AWS Nuke support... "

mkdir -p ${builddir} > /dev/null 2>&1

curl -s https://awspolicygen.s3.amazonaws.com/js/policies.js \
    -o ${builddir}/policies.js

cat ${builddir}/policies.js | sed 's/app.PolicyEditorConfig=//' > ${builddir}/policies.json

rm -rf ${builddir}/aws-nuke
(cd ${builddir} && git clone ${awsnukegit} aws-nuke > /dev/null 2>&1)

cd ${tooldir} 
${toolcmd} -nuke-source-dir=${builddir}/aws-nuke -policies-js-file=${builddir}/policies.json -generate-markdown > ${builddir}/awsnuke-support.md 2>/dev/null

echo -e '\b done.'
