#!/bin/bash
# Generate serverless.yml

function generate() {
  filename=${1:-serverless.yml}
  service=${2:-service}
  stage=${3:-dev}
  region=${4:-us-west-2}
  s3_bucket=${5:-s3_bucket}
  dynamodb_table=${6:-dynamodb_table}
  sqs_enabled=${7:-"true"}
  sqs_queue=${8:-sqs_queue}

  echo "sqs_enabled: ${sqs_enabled}"
  echo "sqs_queue: ${sqs_queue}"

  cat << EOT > ${filename}
service: ${service}

frameworkVersion: '3'

provider:
  name: aws
  runtime: go1.x
  memorySize: 128
  stage: \${opt:stage, '${stage}'}
  region: \${opt:region, '${region}'}
  environment:
    REGION: \${self:provider.region}
    DYNAMODB_TABLE: ${dynamodb_table}
    DYNAMODB_TIME_INDEX: TimeIndex
    S3_BUCKET: ${s3_bucket}
EOT

  if [[ "${sqs_enabled}" == "true" ]]; then
    cat << EOT >> ${filename}
    SQS_QUEUE: ${sqs_queue}
EOT
  fi

  cat << EOT >> ${filename}
  iam:
    role:
      statements:
        - Effect: Allow
          Action:
            - dynamodb:GetItem
            - dynamodb:PutItem
            - dynamodb:UpdateItem
            - dynamodb:DeleteItem
          Resource: "arn:aws:dynamodb:\${self:provider.region}:*:table/\${self:provider.environment.DYNAMODB_TABLE}"
        - Effect: Allow
          Action:
            - dynamodb:Query
            - dynamodb:Scan
          Resource: "arn:aws:dynamodb:\${self:provider.region}:*:table/\${self:provider.environment.DYNAMODB_TABLE}/index/\${self:provider.environment.DYNAMODB_TIME_INDEX}"
        - Effect: Allow
          Action:
            - s3:GetObject
            - s3:DeleteObject
          Resource: "arn:aws:s3::*:\${self:provider.environment.S3_BUCKET}/*"
EOT
  if [[ "${sqs_enabled}" == "true" ]]; then
  cat << EOT >> ${filename}
        - Effect: Allow
          Action:
            - sqs:GetQueueUrl
            - sqs:SendMessage
          Resource: "arn:aws:sqs:\${self:provider.region}:*:\${self:provider.environment.SQS_QUEUE}"
EOT
  fi

  cat << EOT >> ${filename}
  apiGateway:
    shouldStartNameWithService: true

package:
  patterns:
    - '!./**'
    - './bin/**'

functions:
  emailReceive:
    handler: bin/functions/emailReceive
  emailsList:
    handler: bin/api/emails/list
    events:
      - httpApi:
          path: /emails
          method: GET
          authorizer:
            type: aws_iam
  emailsGet:
    handler: bin/api/emails/get
    events:
      - httpApi:
          method: GET
          path: /emails/{messageID}
          authorizer:
            type: aws_iam
  emailsTrash:
    handler: bin/api/emails/trash
    events:
      - httpApi:
          method: POST
          path: /emails/{messageID}/trash
          authorizer:
            type: aws_iam
  emailsUntrash:
    handler: bin/api/emails/untrash
    events:
      - httpApi:
          method: POST
          path: /emails/{messageID}/untrash
          authorizer:
            type: aws_iam
  emailsDelete:
    handler: bin/api/emails/delete
    events:
      - httpApi:
          method: DELETE
          path: /emails/{messageID}
          authorizer:
            type: aws_iam
  emailsCreate:
    handler: bin/api/emails/create
    events:
      - httpApi:
          method: POST
          path: /emails
          authorizer:
            type: aws_iam
  emailsSave:
    handler: bin/api/emails/save
    events:
      - httpApi:
          method: PUT
          path: /emails/{messageID}
          authorizer:
            type: aws_iam
  emailsSend:
    handler: bin/api/emails/send
    events:
      - httpApi:
          method: POST
          path: /emails/{messageID}/send
          authorizer:
            type: aws_iam

resources:
  Resources:
    MailboxDynamoDbTable:
      Type: AWS::DynamoDB::Table
      Properties:
        TableName: \${self:provider.environment.DYNAMODB_TABLE}
        AttributeDefinitions:
          - AttributeName: MessageID
            AttributeType: S
          - AttributeName: TypeYearMonth
            AttributeType: S
          - AttributeName: DateTime
            AttributeType: S
        KeySchema:
          - AttributeName: MessageID
            KeyType: HASH
        ProvisionedThroughput:
          ReadCapacityUnits: 3
          WriteCapacityUnits: 1
        GlobalSecondaryIndexes:
          - IndexName: \${self:provider.environment.DYNAMODB_TIME_INDEX}
            KeySchema:
              - AttributeName: TypeYearMonth
                KeyType: HASH
              - AttributeName: DateTime
                KeyType: RANGE
            Projection:
              ProjectionType: INCLUDE
              NonKeyAttributes:
                - Subject
            ProvisionedThroughput:
              ReadCapacityUnits: 3
              WriteCapacityUnits: 1
EOT
}
