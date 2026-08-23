SHELL := /bin/bash

.PHONY: download
download:
	./script/download.sh

.PHONY: build
build:
	./script/build.sh

.PHONY: build-lambda
build-lambda:
	./script/build.sh --zip-only

.PHONY: clean
clean:
	@rm -rf ./bin

# TF_STATE_BUCKET must be set; the backend is a partial config so the bucket
# name stays out of this repo.
.PHONY: init
init:
	@[ -n "$(TF_STATE_BUCKET)" ] || { echo "TF_STATE_BUCKET is not set"; exit 1; }
	@terraform init -backend-config="bucket=$(TF_STATE_BUCKET)"

.PHONY: plan
plan: clean build-lambda
	@terraform plan

.PHONY: apply
apply: clean build-lambda
	@terraform apply

# Deploys the binaries from the latest release rather than a local build.
.PHONY: deploy
deploy: clean download
	@terraform apply

.PHONY: destroy
destroy:
	@terraform destroy

.PHONY: fmt
fmt:
	@terraform fmt
	@go fmt ./...

.PHONY: test
test:
	@CGO_ENABLED=1 go test -race -covermode=atomic $(shell go list ./... | grep -v /integration)
