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
  auth_method=${9:-iam}

  cat << EOT > ${filename}
service: ${service}

frameworkVersion: '3'

provider:
  name: aws
  runtime: provided.al2
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
        - Effect: Allow
          Action:
            - ses:SendEmail
          Resource: "arn:aws:ses:${self:provider.region}:*:identity/*"
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
  individually: true
EOT

  if [[ "${auth_method}" == "iam" ]]; then
    auth_statement="
          authorizer:
            type: aws_iam"
  else
    auth_statement=""
  fi

  cat << EOT >> ${filename}
functions:
  emailReceive:
    handler: bootstrap
    environment:
      ENABLE_SQS: true
    package:
      artifact: bin/emailReceive.zip
  emailsList:
    handler: bin/api/emails/list
    events:
      - httpApi:
          path: /emails
          method: GET${auth_statement}
    package:
      artifact: bin/list.zip
  emailsGet:
    handler: bootstrap
    events:
      - httpApi:
          method: GET
          path: /emails/{messageID}${auth_statement}
    package:
      artifact: bin/get.zip
  emailsTrash:
    handler: bootstrap
    events:
      - httpApi:
          method: POST
          path: /emails/{messageID}/trash${auth_statement}
    package:
      artifact: bin/trash.zip
  emailsUntrash:
    handler: bootstrap
    events:
      - httpApi:
          method: POST
          path: /emails/{messageID}/untrash${auth_statement}
    package:
      artifact: bin/untrash.zip
  emailsDelete:
    handler: bootstrap
    events:
      - httpApi:
          method: DELETE
          path: /emails/{messageID}${auth_statement}
    package:
      artifact: bin/delete.zip
  emailsCreate:
    handler: bootstrap
    events:
      - httpApi:
          method: POST
          path: /emails${auth_statement}
    package:
      artifact: bin/create.zip
  emailsSave:
    handler: bootstrap
    events:
      - httpApi:
          method: PUT
          path: /emails/{messageID}${auth_statement}
    package:
      artifact: bin/save.zip
  emailsSend:
    handler: bootstrap
    events:
      - httpApi:
          method: POST
          path: /emails/{messageID}/send${auth_statement}
    package:
      artifact: bin/send.zip

resources:
  Resources:
    MailboxDynamoDbTable:
      Type: AWS::DynamoDB::Table
      Properties:
        TableName: ${self:provider.environment.DYNAMODB_TABLE}
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
          - IndexName: ${self:provider.environment.DYNAMODB_TIME_INDEX}
            KeySchema:
              - AttributeName: TypeYearMonth
                KeyType: HASH
              - AttributeName: DateTime
                KeyType: RANGE
            Projection:
              ProjectionType: INCLUDE
              NonKeyAttributes:
                - Subject
                - From
                - To
            ProvisionedThroughput:
              ReadCapacityUnits: 3
              WriteCapacityUnits: 1
EOT
}
