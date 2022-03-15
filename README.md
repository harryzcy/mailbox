# Mailbox

[![Actions Status](https://github.com/harryzcy/mailbox/workflows/Go/badge.svg)](https://github.com/harryzcy/mailbox/actions)
[![codecov](https://codecov.io/gh/harryzcy/mailbox/branch/main/graph/badge.svg)](https://codecov.io/gh/harryzcy/mailbox)
[![Go Report Card](https://goreportcard.com/badge/github.com/harryzcy/mailbox)](https://goreportcard.com/report/github.com/harryzcy/mailbox)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat)](http://makeapullrequest.com)

Mailbox is a serverless application focused on receiving emails and triggering events. It runs on AWS services, including SES, Lambda, API Gateway, DynamoDB, and SQS.

## Usage

1. Clone the repository.

    ```shell
    git clone https://github.com/harryzcy/mailbox
    ```

1. Install [serverless](https://github.com/serverless/serverless).

    ```shell
    npm install -g serverless
    ```

1. Create an IAM user.

    Create an IAM user with **AdministratorAccess** and export the access key as environment variables.

    ```shell
    export AWS_ACCESS_KEY_ID=<your-key-here>
    export AWS_SECRET_ACCESS_KEY=<your-secret-key-here>
    ```

    For more details, follow [this guide](https://www.serverless.com/framework/docs/providers/aws/guide/credentials).

1. Create an configuration file.

    ```shell
    cp serverless.yml.example serverless.yml
    ```

1. Setup S3, SES, and SQS.

    Manually create S3 buckets, and setup SES and SQS services from AWS console. Put S3 bucket name and SQS queue name in `serverless.yml`.

1. Deploy the app.

    ```shell
    make deploy
    ```

1. Configure email receiving.

    From AWS console -> Configuration -> Email receiving -> Create rule set -> Create rule, add two actions:

    1. Deliver to Amazon S3 bucket, then enter your bucket name.
    2. Invoke AWS Lambda function, and select `mailbox-dev-emailReceive` or `mailbox-prod-emailReceive`.
