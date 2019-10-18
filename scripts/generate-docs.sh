#!/usr/bin/env bash

# Check to make sure mkdocs is installed...

MKDOCS_CMD=mkdocs
MKDOCS_OPTS=-c -s -d html-docs

if [ ! -x "$(command -v ${MKDOCS_CMD})" ]; then
  echo "Error: ${MKDOCS_CMD} not found. Please install ${MKDOCS_CMD} first." >&2
  echo "See https://docs.readthedocs.io/en/stable/intro/getting-started-with-mkdocs.html for more details." >&2
  exit 1
fi

echo "Generating docs..."

${MKDOCS_CMD} build ${MKDOCS_OPTS} 

EXIT_CODE=$?

echo "Done."

exit ${EXIT_CODE}