package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/harryzcy/mailbox/internal/api"
	"github.com/harryzcy/mailbox/internal/email"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/util/apiutil"
)

type createClient struct {
	dynamodbSvc *dynamodb.Client
	sesv2Svd    *sesv2.Client
}

func (c createClient) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return c.dynamodbSvc.GetItem(ctx, params, optFns...)
}

func (c createClient) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	return c.dynamodbSvc.PutItem(ctx, params, optFns...)
}

func (c createClient) BatchWriteItem(ctx context.Context, params *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error) {
	return c.dynamodbSvc.BatchWriteItem(ctx, params, optFns...)
}

func (c createClient) SendEmail(ctx context.Context, params *sesv2.SendEmailInput, optFns ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
	return c.sesv2Svd.SendEmail(ctx, params, optFns...)
}

func (c createClient) TransactWriteItems(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	return c.dynamodbSvc.TransactWriteItems(ctx, params, optFns...)
}

func newCreateClient(cfg aws.Config) createClient {
	return createClient{
		dynamodbSvc: dynamodb.NewFromConfig(cfg),
		sesv2Svd:    sesv2.NewFromConfig(cfg),
	}
}

func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (apiutil.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	fmt.Println("request received")

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(env.Region))
	if err != nil {
		fmt.Printf("unable to load SDK config, %v\n", err)
		return apiutil.NewErrorResponse(400, "invalid input"), nil
	}

	if req.Body == "" {
		fmt.Printf("body is empty\n")
		return apiutil.NewErrorResponse(http.StatusBadRequest, "invalid input"), nil
	}

	input := email.CreateInput{}
	err = json.Unmarshal([]byte(req.Body), &input)
	if err != nil {
		fmt.Printf("failed to unmarshal: %v\n", err)
		return apiutil.NewErrorResponse(http.StatusInternalServerError, "internal error"), nil
	}

	if input.GenerateText == "" {
		input.GenerateText = "auto"
	}
	if (input.GenerateText != "on") && (input.GenerateText != "off") && (input.GenerateText != "auto") {
		fmt.Printf("invalid generateText: %v\n", input.GenerateText)
		return apiutil.NewErrorResponse(http.StatusBadRequest, "invalid input"), nil
	}

	client := newCreateClient(cfg)
	result, err := email.Create(ctx, client, input)
	if err != nil {
		if err == api.ErrInvalidInput {
			return apiutil.NewErrorResponse(http.StatusBadRequest, "invalid input"), nil
		}
		if err == api.ErrTooManyRequests {
			fmt.Println("too many requests")
			return apiutil.NewErrorResponse(http.StatusTooManyRequests, "too many requests"), nil
		}

		fmt.Printf("email create failed: %v\n", err)
		return apiutil.NewErrorResponse(http.StatusInternalServerError, "internal error"), nil
	}

	body, err := json.Marshal(result)
	if err != nil {
		fmt.Printf("marshal failed: %v\n", err)
		return apiutil.NewErrorResponse(http.StatusInternalServerError, "internal error"), nil
	}
	return apiutil.NewSuccessJSONResponse(string(body)), nil
}

func main() {
	lambda.Start(handler)
}
