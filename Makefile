.PHONY: build clean

all: test build

default: test build

fmt: 
	go fmt ./...

format: fmt

vendor: 
	go mod vendor

mod: 
	-go mod init github.com/Optum/dce

vet:
	go vet

lint:
	./scripts/lint.sh

test: mod lint
	./scripts/test.sh

test_functional:
	./scripts/test_functional.sh

build:
	./scripts/build.sh

generate:
	go generate ./...

# deploy builds and deploys go code
# to Lamdbas and CodeBuilds in AWS.
# Before running this command, you will need
deploy: clean build
	cd modules && \
	ns=$$(terraform output namespace) && \
	bucket=$$(terraform output artifacts_bucket_name) && \
	cd .. && \
	./scripts/deploy.sh bin/build_artifacts.zip $${ns} $${bucket}


# `make documentation`
#
# Generates DCE docs
#
# This repo uses [MkDocs](https://www.mkdocs.org/) to generate and serve documentation.
#
# Before running this make command, you must first:
#
# - Install [Python](https://www.python.org/downloads/)
# - Install [npm v5.2+](https://www.npmjs.com/get-npm)
# - Run `pip install -r ./requirements.txt` to install MkDocs
#
# To generate and serve docs, run:
#
# ```.env
# make documentation
# mkdocs serve
# ```
#
# This will serve the documentation at http://127.0.0.1:8000/
#
# Public-facing docs are served by readthedocs.io
documentation:
	cp -f CONTRIBUTING.md ./docs/CONTRIBUTING.md > /dev/null
	cp -f CHANGELOG.md ./docs/CHANGELOG.md > /dev/null
	cp -f CODE_OF_CONDUCT.md ./docs/CODE_OF_CONDUCT.md > /dev/null
	cp -f INDIVIDUAL_CONTRIBUTOR_LICENSE.md ./docs/INDIVIDUAL_CONTRIBUTOR_LICENSE.md > /dev/null
	./scripts/generate-docs.sh

# Serve the documentation locally
serve_docs:
	mkdocs serve


install:
	go install 

clean:
	rm -rf ./bin ./vendor ./html-doc ./site