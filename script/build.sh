#!/bin/bash

apiFuncs=("list" "get" "trash" "untrash" "delete" "create" "save" "send")

for i in "${!apiFuncs[@]}";
do
  func="${apiFuncs[$i]}"
  env GOOS=linux go build -ldflags="-s -w" -o bin/api/emails/"${func}" api/emails/"${func}"/*
done

env GOOS=linux go build -ldflags="-s -w" -o bin/functions/emailReceive functions/emailReceive/*
