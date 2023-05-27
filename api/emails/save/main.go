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
	"github.com/harryzcy/mailbox/internal/email"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/util/apiutil"
)

type saveClient struct {
	dynamodbSvc *dynamodb.Client
	sesv2Svc    *sesv2.Client
}

func (c saveClient) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return c.dynamodbSvc.GetItem(ctx, params, optFns...)
}

func (c saveClient) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	return c.dynamodbSvc.PutItem(ctx, params, optFns...)
}

func (c saveClient) TransactWriteItems(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	return c.dynamodbSvc.TransactWriteItems(ctx, params, optFns...)
}

func (c saveClient) SendEmail(ctx context.Context, params *sesv2.SendEmailInput, optFns ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
	return c.sesv2Svc.SendEmail(ctx, params, optFns...)
}

func newSaveClient(cfg aws.Config) saveClient {
	return saveClient{
		dynamodbSvc: dynamodb.NewFromConfig(cfg),
		sesv2Svc:    sesv2.NewFromConfig(cfg),
	}
}

func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (apiutil.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	fmt.Println("request received")

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(env.Region))
	if err != nil {
		fmt.Printf("unable to load SDK config, %v\n", err)
		return apiutil.NewErrorResponse(http.StatusInternalServerError, "internal error"), nil
	}

	messageID := req.PathParameters["messageID"]
	fmt.Printf("request params: [messagesID] %s\n", messageID)

	if req.Body == "" {
		fmt.Printf("body is empty\n")
		return apiutil.NewErrorResponse(http.StatusBadRequest, "invalid input"), nil
	}

	input := email.SaveInput{}
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

	input.MessageID = messageID
	client := newSaveClient(cfg)
	result, err := email.Save(ctx, client, input)
	if err != nil {
		if err == email.ErrInvalidInput {
			return apiutil.NewErrorResponse(http.StatusBadRequest, "invalid input"), nil
		}
		if err == email.ErrTooManyRequests {
			fmt.Println("too many requests")
			return apiutil.NewErrorResponse(http.StatusTooManyRequests, "too many requests"), nil
		}

		fmt.Printf("email save failed: %v\n", err)
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
