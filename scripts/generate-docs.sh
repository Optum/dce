#!/usr/bin/env bash

# Check to make sure swagger-markdown is installed, which will be used to generate
# the api-documentation.md file from the Swagger YAML

NPX_CMD=npx
NPX_OPTS=swagger-markdown
SWAGGER_YAML=modules/swaggerRedbox.yaml
API_MARKDOWN=docs/api-documentation.md

if [ ! -x "$(command -v ${NPX_CMD})" ]; then
  echo "Error: ${NPX_CMD} not found, which is used to execute \"swagger-markdown\". Please install the \"${NPX_CMD}\" first." >&2
  exit 1
fi

# Now check to make sure that Swagger file is there...

if [ ! -f ${SWAGGER_YAML} ]; then
  echo "Error: cannot find the \"${SWAGGER_YAML}\" file, which is expected for generating API documentation." >&2
  echo "Please check the path." >&2
  exit 1
fi

echo "Generating API documentation..."
${NPX_CMD} ${NPX_OPTS} -i ${SWAGGER_YAML} -o ${API_MARKDOWN}
echo "Finished generating API documentation."

# Check to make sure mkdocs is installed...
MKDOCS_CMD=mkdocs
MKDOCS_OPTS="-c -s -d html-docs"

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