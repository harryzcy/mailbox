# Mailbox

[![Tests](https://github.com/harryzcy/mailbox/actions/workflows/test.yml/badge.svg)](https://github.com/harryzcy/mailbox/actions)
[![codecov](https://codecov.io/gh/harryzcy/mailbox/branch/main/graph/badge.svg)](https://codecov.io/gh/harryzcy/mailbox)
[![Go Report Card](https://goreportcard.com/badge/github.com/harryzcy/mailbox)](https://goreportcard.com/report/github.com/harryzcy/mailbox)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat)](http://makeapullrequest.com)
[![License: MIT](https://img.shields.io/github/license/harryzcy/mailbox)](https://opensource.org/licenses/MIT)

Docs: [English](README.md) • [简体中文](README_zh.md)

Versatile email infrastructure that operates on AWS serverless platform.

## Table of Contents

- [Mailbox](#mailbox)
  - [Table of Contents](#table-of-contents)
  - [Usage](#usage)
  - [API](#api)
  - [Clients](#clients)
    - [Web](#web)
    - [CLI](#cli)
  - [Architecture](#architecture)
  - [Contributing](#contributing)
    - [Development environment](#development-environment)

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

1. Setup AWS services.

    Manually create S3 buckets, and setup SES and SQS (optional) from AWS console.

1. Copy over example configurations and fill in correct fields.

    ```shell
    cp serverless.yml.example serverless.yml
    ```

    Under `provider.environment` section, modify `REGION`, `S3_BUCKET`, `SQS_QUEUE` (optional, only if SQS should be enabled).

1. Deploy the app.

    ```shell
    make deploy
    ```

1. Configure email receiving.

    From AWS console -> Configuration -> Email receiving -> Create rule set -> Create rule, add two actions:

    1. Deliver to Amazon S3 bucket, then enter your bucket name.
    2. Invoke AWS Lambda function, and select `mailbox-dev-emailReceive` or `mailbox-prod-emailReceive`.

## API

See [doc/API.md](doc/api.md)

## Clients

### Web

See [mailbox-browser](https://github.com/harryzcy/mailbox-browser).

| Dark mode |  Light mode |
|:---------:|:-----------:|
| ![Screenshot Dark Mode](https://github.com/harryzcy/mailbox-browser/assets/37034805/b77a6c40-c6c1-4dd8-98de-2add697b26f9) | ![Screenshot Light Mode](https://github.com/harryzcy/mailbox-browser/assets/37034805/ce9ab42c-923a-4b03-8ee4-bcdc9d4b72ed) |

### CLI

```bash
go install github.com/harryzcy/mailbox-cli
```

For details, refer to [mailbox-cli](https://github.com/harryzcy/mailbox-cli)

## Architecture

It runs on AWS services, including SES, Lambda, API Gateway, DynamoDB, and SQS.

![Architecture](./doc/architecture.svg)

## Contributing

### Development environment

- Go >= 1.21

Note that only the two most recent minor versions of Go are officially supported.
