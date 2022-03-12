.PHONY: build clean deploy

build:
	export GO111MODULE=on
	env GOOS=linux go build -ldflags="-s -w" -o bin/functions/email-receive functions/email-receive/*
	env GOOS=linux go build -ldflags="-s -w" -o bin/api/emails/list api/emails/list/*
	env GOOS=linux go build -ldflags="-s -w" -o bin/api/emails/get api/emails/get/*
	env GOOS=linux go build -ldflags="-s -w" -o bin/api/emails/trash api/emails/trash/*
	env GOOS=linux go build -ldflags="-s -w" -o bin/api/emails/delete api/emails/delete/*

clean:
	rm -rf ./bin ./vendor

deploy: clean build
	sls deploy --verbose

remove: clean
	sls remove --verbose
