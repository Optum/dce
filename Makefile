.PHONY: build clean

all: test build

default: test build

fmt:
	go fmt ./...

format: fmt

vendor:
	go mod vendor

mod:
	go mod tidy -v

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
	ns=rebura-dce && \
	bucket=dce-artifacts-rebura-dce && \
	cd .. && \
	./scripts/deploy.sh bin/build_artifacts.zip $${ns} $${bucket}


# `make documentation`
#
# Generates DCE docs as HTML
# in the /docs/_build/html directory
#
# This repo uses [Sphinx](http://www.sphinx-doc.org/en/master/) to generate documentation from markdown files
#
# Before running this make command, you must first:
#
# - Install [Python v3+](https://www.python.org/downloads/)
# - Run `pip install -r ./docs/requirements.txt` to install Sphinx
#
documentation:
	./scripts/generate-awsnuke-docs.sh
	cd docs && make html

# Serve the documentation locally
# Uses https://pypi.org/project/sphinx-autobuild/
#
# Before running this make command, you must first:
#
# - Install [Python v3+](https://www.python.org/downloads/)
# - Run `pip install -r ./docs/requirements.txt` to install Sphinx
serve_docs: documentation
	cd docs && make livehtml


install:
	go install

clean:
	rm -rf ./bin ./vendor ./html-doc ./site

setup:
	./scripts/install_ci.sh
