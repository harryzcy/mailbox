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

.PHONY: deploy
deploy: clean download
	@sls deploy --verbose

.PHONY: build-deploy
build-deploy: clean build
	@sls deploy --verbose

.PHONY: remove
remove: clean
	@sls remove --verbose

.PHONY: test
test:
	@CGO_ENABLED=1 go test -race -covermode=atomic $(shell go list ./... | grep -v /integration)
