# Mailbox

[![Tests](https://github.com/harryzcy/mailbox/actions/workflows/test.yml/badge.svg)](https://github.com/harryzcy/mailbox/actions)
[![codecov](https://codecov.io/gh/harryzcy/mailbox/branch/main/graph/badge.svg)](https://codecov.io/gh/harryzcy/mailbox)
[![Go Report Card](https://goreportcard.com/badge/github.com/harryzcy/mailbox)](https://goreportcard.com/report/github.com/harryzcy/mailbox)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat)](http://makeapullrequest.com)
[![License: MIT](https://img.shields.io/github/license/harryzcy/mailbox)](https://opensource.org/licenses/MIT)

文档: [English](README.md) • [简体中文](README_zh.md)

Mailbox 是一个接收邮件、触发消息通知的无服务应用。

目前运行在 AWS 服务上，使用 SES, Lambda, API Gateway, DynamoDB, 和 SQS。

## 客户端

### Web

见 [mailbox-browser](https://github.com/harryzcy/mailbox-browser)

| Dark mode |  Light mode |
|:---------:|:-----------:|
| ![Screenshot Dark Mode](https://github.com/harryzcy/mailbox-browser/assets/37034805/b77a6c40-c6c1-4dd8-98de-2add697b26f9) | ![Screenshot Light Mode](https://github.com/harryzcy/mailbox-browser/assets/37034805/ce9ab42c-923a-4b03-8ee4-bcdc9d4b72ed) |

### CLI

```bash
go install github.com/harryzcy/mailbox-cli
```

细节参见 [mailbox-cli](https://github.com/harryzcy/mailbox-cli)

## 部署

1. Clone 仓库.

    ```shell
    git clone https://github.com/harryzcy/mailbox
    ```

1. 安装 [serverless](https://github.com/serverless/serverless).

    ```shell
    npm install -g serverless
    ```

1. 创建一个 IAM 用户.

    创建一个 IAM 用户并赋予 **AdministratorAccess** 权限，把 access key 设为 environment variables.

    ```shell
    export AWS_ACCESS_KEY_ID=<your-key-here>
    export AWS_SECRET_ACCESS_KEY=<your-secret-key-here>
    ```

    更多细节参考 [serverless 文档](https://www.serverless.com/framework/docs/providers/aws/guide/credentials).

1. 设置 AWS 服务.

    在 AWS 控制台中创建 S3 存储桶，SES 服务 和 SQS 队列 (可选)。

1. 复制 serverless 配置。

    ```shell
    cp serverless.yml.example serverless.yml
    ```

    在 `provider.environment` 下, 修改 `REGION`, `S3_BUCKET`, `SQS_QUEUE` (可选, 使用 SQS 才需要).

1. 部署应用.

    ```shell
    make deploy
    ```

1. 设置邮件接收.

    在 AWS console -> Configuration -> Email receiving -> Create rule set -> Create rule 中, 添加两条 Action 策略:

    1. Deliver to Amazon S3 bucket，然后填入存储桶名称.
    2. Invoke AWS Lambda function，然后选择 `mailbox-dev-emailReceive` 或 `mailbox-prod-emailReceive`.

1. 部署 [mailbox-browser](https://github.com/harryzcy/mailbox-browser) 或者使用 [mailbox-cli](https://github.com/harryzcy/mailbox-cli).

## API

见 [doc/API.md](doc/api.md)

## 架构

![Architecture](./doc/architecture.svg)

## Contributing

### 开发环境

- Go >= 1.21
