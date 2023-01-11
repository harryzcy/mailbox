#!/bin/bash

BUILD_COMMIT=$(git rev-parse --short HEAD)
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILD_VERSION=$(git describe --tags --always)

ENVIRONMENT="env GOOS=linux GOARCH=amd64"

apiFuncs=("list" "get" "getRaw" "getContent" "read" "trash" "untrash" "delete" "create" "save" "send")

for i in "${!apiFuncs[@]}";
do
  func="${apiFuncs[$i]}"
  ${ENVIRONMENT} go build -ldflags="-s -w" -o bin/api/emails/"${func}" api/emails/"${func}"/*
  cp bin/api/emails/"${func}" bin/bootstrap
  zip -j bin/"${func}".zip bin/bootstrap
done

${ENVIRONMENT} go build -ldflags="-s -w \
                                  -X 'main.version=${BUILD_VERSION}' \
                                  -X 'main.commit=${BUILD_COMMIT}' \
                                  -X 'main.buildDate=${BUILD_DATE}' \
                                  " \
                        -o bin/api/info api/info/*
cp bin/api/info bin/bootstrap
zip -j bin/info.zip bin/bootstrap

${ENVIRONMENT} go build -ldflags="-s -w" -o bin/functions/emailReceive functions/emailReceive/*
cp bin/functions/emailReceive bin/bootstrap
zip -j bin/emailReceive.zip bin/bootstrap
rm bin/bootstrap
