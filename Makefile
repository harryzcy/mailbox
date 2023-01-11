.PHONY: build
build:
	./script/build.sh

.PHONY: clean
clean:
	rm -rf ./bin ./vendor

.PHONY: deploy
deploy: clean build
	sls deploy --verbose

.PHONY: remove
remove: clean
	sls remove --verbose

.PHONY: test
test:
	go test -race -covermode=atomic ./...
