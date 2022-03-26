.PHONY: build clean deploy remove test

build:
	export GO111MODULE=on
	env GOOS=linux go build -ldflags="-s -w" -o bin/functions/emailReceive functions/emailReceive/*
	env GOOS=linux go build -ldflags="-s -w" -o bin/api/emails/list api/emails/list/*
	env GOOS=linux go build -ldflags="-s -w" -o bin/api/emails/get api/emails/get/*
	env GOOS=linux go build -ldflags="-s -w" -o bin/api/emails/trash api/emails/trash/*
	env GOOS=linux go build -ldflags="-s -w" -o bin/api/emails/untrash api/emails/untrash/*
	env GOOS=linux go build -ldflags="-s -w" -o bin/api/emails/delete api/emails/delete/*
	env GOOS=linux go build -ldflags="-s -w" -o bin/api/emails/create api/emails/create/*
	env GOOS=linux go build -ldflags="-s -w" -o bin/api/emails/save api/emails/save/*
	env GOOS=linux go build -ldflags="-s -w" -o bin/api/emails/send api/emails/send/*

clean:
	rm -rf ./bin ./vendor

deploy: clean build
	sls deploy --verbose

remove: clean
	sls remove --verbose

test:
	go test -race -covermode=atomic ./...
