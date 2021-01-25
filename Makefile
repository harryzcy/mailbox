.PHONY: build clean deploy gomodgen

build: gomodgen
	export GO111MODULE=on
	env GOOS=linux go build -ldflags="-s -w" -o bin/email-receive email-receive/*
	env GOOS=linux go build -ldflags="-s -w" -o bin/api/emails/list api/emails/list/*

clean:
	rm -rf ./bin ./vendor go.sum

deploy: clean build
	sls deploy --verbose

remove: clean
	sls remove --verbose

gomodgen:
	chmod u+x gomod.sh
	./gomod.sh
