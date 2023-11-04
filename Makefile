.PHONY: build
build:
	./script/build.sh

.PHONY: clean
clean:
	rm -r ./bin

.PHONY: deploy
deploy: clean build
	sls deploy --verbose

.PHONY: remove
remove: clean
	sls remove --verbose

.PHONY: test
test:
	go test -race -covermode=atomic ./...
