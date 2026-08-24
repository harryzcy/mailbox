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

# An unset required variable makes terraform wait on stdin, stranding the state
# lock if the run is interrupted. -input=false errors instead of asking, and
# doesn't affect the apply approval prompt.
.PHONY: require-vars
require-vars:
	@[ -n "$(TF_VAR_aws_s3_bucket_override)" ] || { echo "TF_VAR_aws_s3_bucket_override is not set"; exit 1; }

# Previews what deploy would do, so it uses the same released binaries.
.PHONY: plan
plan: require-vars download
	@terraform plan -input=false

# Deploys the binaries built from the working tree rather than the latest release.
.PHONY: apply
apply: require-vars clean build-lambda
	@terraform apply -input=false

.PHONY: deploy
deploy: require-vars download
	@terraform apply -input=false

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
