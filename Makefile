.PHONY: build clean deploy remove test

build:
	./script/build.sh

clean:
	rm -rf ./bin ./vendor

deploy: clean build
	sls deploy --verbose

remove: clean
	sls remove --verbose

test:
	go test -race -covermode=atomic ./...
