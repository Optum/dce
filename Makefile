.PHONY: build clean

all: test build

default: test build

fmt: 
	go fmt ./...

format: fmt

vendor: 
	go mod vendor

mod: 
	-go mod init github.com/Optum/Redbox

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

release:
	./scripts/release.sh \
		--site_url github.optum.com \
		--github_org CommercialCloud-Team \
		--repository aws_redbox \
		--artifacts bin/build_artifacts.zip,bin/terraform_artifacts.zip,scripts/deploy.sh,scripts/restore_db.sh \
		--tag $(TAG)


deploy_local:
	cd modules && terraform init \
	&& terraform apply -input=false -auto-approve -var-file=local_terraform.tfvars

destroy_local:
	cd modules && terraform init \
	&& terraform destroy -force -var-file=local_terraform.tfvars

install:
	go install 

clean:
	rm -rf ./bin ./vendor