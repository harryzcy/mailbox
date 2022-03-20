#!/bin/bash

current=$(cd -P -- "$(dirname -- "$0")" && pwd -P)
base_dir=$(dirname ${current})
cd ${base_dir}

cp serverless.yml serverless.yml.example
sed -i '' 's/S3_BUCKET: your-mailbox/S3_BUCKET: example-mailbox # set this to your S3 bucket name/' serverless.yml.example
sed -i '' 's/SQS_QUEUE: your-mailbox/SQS_QUEUE: example-mailbox # set this to your SQS queue name/' serverless.yml.example
