#!/bin/bash

current=$(cd -P -- "$(dirname -- "$0")" && pwd -P)
base_dir=$(dirname ${current})
cd ${base_dir}

# parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    -s|--serverless)
      SERVERLESS_FILE="$2"
      shift # past argument
      shift # past value
      ;;
    -y|--yes)
      YES=true
      shift # past argument
      ;;
    -*|--*)
      echo "Unknown option $1"
      exit 1
      ;;
  esac
done

# default config files
if [ -z "${SERVERLESS_FILE}" ]; then
  SERVERLESS_FILE="serverless.yml"
fi

if [ -z "${YES}" ]; then
  YES=false
fi

echo "Setting up config files..."

# service
if [[ "${YES}" != "true" ]]; then
  read -p "Enter service name (mailbox): " service
fi
if [ -z "${service}" ]; then
    service="mailbox"
fi

# stage
if [[ "${YES}" != "true" ]]; then
  read -p "Enter stage (dev): " stage
fi
if [ -z "${stage}" ]; then
    stage="dev"
fi

# region
if [[ "${YES}" != "true" ]]; then
  read -p "Enter region (us-west-2): " region
fi
if [ -z "${region}" ]; then
    region="us-west-2"
fi

# S3_BUCKET
if [[ "${YES}" != "true" ]]; then
  read -p "Enter S3 bucket name (\$service): " s3_bucket
fi
if [ -z "${s3_bucket}" ]; then
    s3_bucket="${service}"
fi

# DYNAMODB_TABLE
if [[ "${YES}" != "true" ]]; then
  read -p "Enter DynamoDB table name (\$serivice-\$stage): " dynamodb_table
fi
if [ -z "${dynamodb_table}" ]; then
    dynamodb_table="${service}-\${self:provider.stage}"
fi

# Enable SQS
if [[ "${YES}" != "true" ]]; then
  read -p "Enable SQS? (Y/n): " sqs_enabled
fi
if [ -z "${sqs_enabled}" ]; then
    sqs_enabled="y"
fi
case ${sqs_enabled} in
  [Yy]* ) sqs_enabled="true" ;;
  [Nn]* ) sqs_enabled="false" ;;
  * ) sqs_enabled="invalid" ;;
esac
if [[ "${sqs_enabled}" == "invalid" ]]; then
  echo "Invalid option"
  exit 1
fi

# SQS_QUEUE
if [[ "${sqs_enabled}" == "true" ]]; then
  if [[ "${YES}" != "true" ]]; then
    read -p "Enter SQS queue name (\$service-\$stage): " sqs_queue
  fi
  if [ -z "${sqs_queue}" ]; then
    sqs_queue="${service}-\${self:provider.stage}"
  fi
else
  sqs_queue=""
fi

# Auth method
if [[ "${YES}" != "true" ]]; then
  read -p "Enter auth method (iam (default) | none): " auth_method
fi
if [ -z "${auth_method}" ]; then
    auth_method="iam"
fi
case ${auth_method} in
    [iI][aA][mM]|[iI] ) auth_method="iam" ;;
    [nN][oO][nN][eE]|[nN] ) auth_method="none" ;;
    * ) auth_method="invalid" ;;
esac
if [[ "${auth_method}" == "invalid" ]]; then
    echo "Invalid auth method"
    exit 1
fi

echo
echo "About to write to ${SERVERLESS_FILE}:"
echo

echo "service: ${service}"
echo "stage: ${stage}"
echo "region: ${region}"
echo "S3 bucket name: ${s3_bucket}"
echo "DynamoDB table name: ${dynamodb_table}"
echo "SQS enabled: ${sqs_enabled}"
if [[ "${sqs_enabled}" == "true" ]]; then
    echo "SQS queue name: ${sqs_queue}"
fi
echo "Auth method: ${auth_method}"

echo
if [[ "${YES}" != "true" ]]; then
  read -p "Is this correct? (Y/n): " correct
fi
if [[ -z "$correct" ]]; then
    correct="y"
fi

case "$correct" in
    [yY][eE][sS]|[yY]) 
      ;;
    *)
      echo "Aborted."
      exit 1
      ;;
esac

source ./script/generate_serverless.sh

generate "${SERVERLESS_FILE}" \
         "${service}" \
         "${stage}" \
         "${region}" \
         "${s3_bucket}" \
         "${dynamodb_table}" \
         "${sqs_enabled}" \
         "${sqs_queue}" \
          "${auth_method}"

echo "Done."
echo
echo "Please review the config files and run 'make deploy' to deploy."
