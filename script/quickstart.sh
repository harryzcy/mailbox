#!/bin/bash

current=$(cd -P -- "$(dirname -- "$0")" && pwd -P)
base_dir=$(dirname ${current})
cd ${base_dir}

# parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    -f|--file)
      CONFIG_FILE="$2"
      shift # past argument
      shift # past value
      ;;
    -*|--*)
      echo "Unknown option $1"
      exit 1
      ;;
  esac
done

# default config file is 'serverless.yml'
if [ -z "${CONFIG_FILE}" ]; then
  CONFIG_FILE="serverless.yml"
fi


echo "Setting up config files..."

# stage
read -p "Enter stage (dev): " stage
if [ -z "${stage}" ]; then
    stage="dev"
fi

# region
read -p "Enter region (us-west-2): " region
if [ -z "${region}" ]; then
    region="us-west-2"
fi

# S3_BUCKET
while [[ -z "${s3_bucket}" ]]; do
    read -p "Enter S3 bucket name: " s3_bucket
done

# SQS_QUEUE
while [[ -z "${sqs_queue}" ]]; do
    read -p "Enter SQS queue name: " sqs_queue
done

echo "About to write to ${CONFIG_FILE}:"
echo

echo "stage: ${stage}"
echo "region: ${region}"
echo "S3 bucket name: ${s3_bucket}"
echo "SQS queue name: ${sqs_queue}"

echo
read -p "Is this correct? (Y/n)" answer

if [[ -z "${answer}" || "${answer}" == "y" ]]; then
    answer="Y"
fi

if [[ "${answer}" != "Y" ]]; then
    echo "Aborted."
    exit 1
fi

cp serverless.yml.example ${CONFIG_FILE}
perl -i -pe"s/dev/${stage}/g" ${CONFIG_FILE}
perl -i -pe"s/us-west-2/${region}/g" ${CONFIG_FILE}
perl -i -pe"s/example-mailbox # set this to your S3 bucket name/${s3_bucket}/g" ${CONFIG_FILE}
perl -i -pe"s/example-mailbox # set this to your SQS queue name/${sqs_queue}/g" ${CONFIG_FILE}

echo "Done."