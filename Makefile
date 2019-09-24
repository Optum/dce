.PHONY: build clean

all: test build

default: test build

fmt: 
	go fmt ./...

format: fmt

vendor: 
	go mod vendor

mod: 
	-go mod init github.com/Optum/Dcs

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

install:
	go install 

clean:
	rm -rf ./bin ./vendor