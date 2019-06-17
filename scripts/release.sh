#!/usr/bin/env bash
set -ueo pipefail

# release.sh
# This script will push named artifacts into a named repo for consumption by a deployment script (separate)
# and create a named release for that repo.
# Arguments:
#   -r|--repository - Named repository for artifacts to be pushed to
#   -a/--artifacts - Comma separated list of artifacts for upload (explicit/relative pathing accepted)
#   -t/--tag - Release name to be generated for the upload (this is individual per release, requires increment with each run of this script)
#   -s/-site_url - Github API hostname
#   -o/--github_org - Github Organizaiton
#   
#   Environment Variables:
#   GIT_TOKEN : (required) Github token (This token is required to have push access to the org/repo in question)
# 
#   Example:
#   ./release.sh -r test_release -o my_org -t rel_tag -g 0xxxxxxxxxxxxx -a test.zip,test2.zip,test3.zip
# 
# Required tools on execution host:
# Bourne Again SHell (documented for completeness)
# jq - command line json parser - https://stedolan.github.io/jq/
# curl - command line http client - https://curl.haxx.se/
# 
# This script utilizes the Github Enterprise paths in urls for use with the github API v3.
# https://developer.github.com/v3/

PARAMS=""
while (( "$#" )); do
  case "$1" in
    -s|--site_url)
      SITE=$2
      shift 2
      ;;
    -r|--repository)
      REPO=$2
      shift 2
      ;;
    -o|--github_org)
      ORG=$2
      shift 2
      ;;
    -a|--artifacts)
      ART=$2
      shift 2
      ;;
    -t|--tag)
      VERSION=$2
      shift 2
      ;;
    -*|--*=) # unsupported flags
      echo "Error: Unsupported flag $1" >&2
      exit 1
      ;;
    *) # preserve positional arguments
      PARAMS="$PARAMS $1"
      shift
      ;;
  esac
done

# Assign our env GITHUB_TOKEN var to TOKEN
TOKEN=$GIT_TOKEN

# Define our payload for the release, currently ONLY master
payload=$(printf '{"tag_name": "%s","target_commitish": "master","name": "%s","body": "Release of version %s","draft": false,"prerelease": false}' $VERSION $VERSION $VERSION)

# POST activity to create release
echo "Creating release \"${VERSION}\"..."
response=$(
  curl -X POST \
      --fail \
      --silent \
      --header "Authorization: token ${TOKEN}" \
      --location \
      --data "$payload" \
      "https://${SITE}/api/v3/repos/${ORG}/${REPO}/releases"
      )
echo "Creating release \"${VERSION}\"... done."


echo "Uploading artifacts to release \"${VERSION}\"...."
# Pull the upload_url out of the headers - requires jq
upload_url="$(echo "$response" | jq -r .upload_url | sed -e "s/{?name,label}//")"

# Upload the artifacts to the release
IFS=',' list=($ART)
for item in "${list[@]}"; do
  curl -X POST \
      --header "Authorization: token ${TOKEN}" \
      --header "Content-Type:application/gzip" \
      --silent -o /dev/null \
      --data-binary "@$item" \
      "$upload_url?name=$(basename "$item")"
done
echo "Uploading artifacts to release \"${VERSION}\".... done."


