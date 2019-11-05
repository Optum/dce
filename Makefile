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

deploy_local: build
	./scripts/deploy_local/deploy_local_build.sh

destroy_local:
	./scripts/deploy_local/destroy_local_build.sh

documentation:
	cp -f CONTRIBUTING.md ./docs/CONTRIBUTING.md > /dev/null
	cp -f CHANGELOG.md ./docs/CHANGELOG.md > /dev/null
	cp -f CODE_OF_CONDUCT.md ./docs/CODE_OF_CONDUCT.md > /dev/null
	cp -f INDIVIDUAL_CONTRIBUTOR_LICENSE.md ./docs/INDIVIDUAL_CONTRIBUTOR_LICENSE.md > /dev/null
	./scripts/generate-docs.sh


install:
	go install 

clean:
	rm -rf ./bin ./vendor ./html-doc ./site