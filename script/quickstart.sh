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
    -c|--config)
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

# default config files
if [ -z "${SERVERLESS_FILE}" ]; then
  SERVERLESS_FILE="serverless.yml"
fi

if [ -z "${CONFIG_FILE}" ]; then
  CONFIG_FILE="config.yml"
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

# Enable SQS
while [[ -z "${sqs_enabled}" ]] || [[ "${sqs_enabled}" != "y" && "${sqs_enabled}" != "n" ]]; do
    read -p "Enable SQS? (Y/n): " sqs_enabled
    if [[ -z "${sqs_enabled}" ]] || [[ "${sqs_enabled}" == "Y" ]]; then
        sqs_enabled="y"
    elif [[ "${sqs_enabled}" == "N" ]]; then
        sqs_enabled="n"
    fi
done

function boolean() {
  case $1 in
    TRUE) echo true ;;
    FALSE) echo false ;;
    y) echo true ;;
    n) echo false ;;
    *) echo "Err: Unknown boolean value \"$1\"" 1>&2; exit 1 ;;
   esac
}

echo
echo "About to write to ${SERVERLESS_FILE} and ${CONFIG_FILE}:"
echo

echo "stage: ${stage}"
echo "region: ${region}"
echo "S3 bucket name: ${s3_bucket}"
echo "SQS queue name: ${sqs_queue}"
echo "SQS enabled: $(boolean ${sqs_enabled})"

echo
read -p "Is this correct? (Y/n)" answer

if [[ -z "${answer}" || "${answer}" == "y" ]]; then
    answer="Y"
fi

if [[ "${answer}" != "Y" ]]; then
    echo "Aborted."
    exit 1
fi

cp serverless.yml.example ${SERVERLESS_FILE}
perl -i -pe"s/dev/${stage}/g" ${SERVERLESS_FILE}
perl -i -pe"s/us-west-2/${region}/g" ${SERVERLESS_FILE}
perl -i -pe"s/example-mailbox # set this to your S3 bucket name/${s3_bucket}/g" ${SERVERLESS_FILE}
perl -i -pe"s/example-mailbox # set this to your SQS queue name/${sqs_queue}/g" ${SERVERLESS_FILE}


cat << EOT > ${CONFIG_FILE}
sqs:
  enabled: $(boolean ${sqs_enabled})
EOT

echo "Done."
echo "Please review the config files and run 'make deploy' to deploy."
