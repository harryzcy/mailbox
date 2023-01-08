#!/bin/bash

apiFuncs=("list" "get" "getContent" "read" "trash" "untrash" "delete" "create" "save" "send")

for i in "${!apiFuncs[@]}";
do
  func="${apiFuncs[$i]}"
  env GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/api/emails/"${func}" api/emails/"${func}"/*
  cp bin/api/emails/"${func}" bin/bootstrap
  zip -j bin/"${func}".zip bin/bootstrap
done

env GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/functions/emailReceive functions/emailReceive/*
cp bin/functions/emailReceive bin/bootstrap
zip -j bin/emailReceive.zip bin/bootstrap
rm bin/bootstrap
