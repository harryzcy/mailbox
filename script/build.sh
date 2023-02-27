#!/bin/bash

BUILD_COMMIT=$(git rev-parse --short HEAD)
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILD_VERSION=$(git describe --tags --always)

ENVIRONMENT="env GOOS=linux GOARCH=amd64"

apiFuncs=(
  "emails/list" "emails/get" "emails/getRaw" "emails/getContent" "emails/read" "emails/trash" "emails/untrash" "emails/delete" "emails/create" "emails/save" "emails/send"
  "threads/get"
  "info"
)

mkdir -p bin/emails
mkdir -p bin/threads

for i in "${!apiFuncs[@]}";
do
  func="${apiFuncs[$i]}"
  ${ENVIRONMENT} go build -ldflags="-s -w" -o bin/api/"${func}" api/"${func}"/*
  cp bin/api/"${func}" bin/bootstrap
  zip -j bin/"${func}".zip bin/bootstrap
done

${ENVIRONMENT} go build -ldflags="-s -w" -o bin/functions/emailReceive functions/emailReceive/*
cp bin/functions/emailReceive bin/bootstrap
zip -j bin/emailReceive.zip bin/bootstrap
rm bin/bootstrap
