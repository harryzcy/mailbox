#!/bin/bash

BUILD_COMMIT=$(git rev-parse --short HEAD)
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILD_VERSION=$(git describe --tags --always)

ENVIRONMENT="env GOOS=linux GOARCH=amd64 CGO_ENABLED=0"

apiFuncs=(
  "emails/list" "emails/get" "emails/getRaw" "emails/getContent" "emails/read" "emails/trash" "emails/untrash" "emails/delete" "emails/create" "emails/save" "emails/send"
  "threads/get" "threads/trash" "threads/untrash" "threads/delete"
)

for i in "${!apiFuncs[@]}"; do
  func="${apiFuncs[$i]}"
  ${ENVIRONMENT} go build -ldflags="-s -w" -o bin/api/"${func}" api/"${func}"/*
  cp bin/api/"${func}" bin/bootstrap
  zipFilename="${func//\//_}"
  zip -j bin/"${zipFilename}".zip bin/bootstrap
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
