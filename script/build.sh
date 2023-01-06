#!/bin/bash

apiFuncs=("list" "get" "getContent" "trash" "untrash" "delete" "create" "save" "send")

for i in "${!apiFuncs[@]}";
do
  func="${apiFuncs[$i]}"
  env GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/api/emails/"${func}" api/emails/"${func}"/*
done

env GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/functions/emailReceive functions/emailReceive/*
