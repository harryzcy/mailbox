package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/platform"
	"github.com/harryzcy/mailbox/internal/thread"
	"github.com/harryzcy/mailbox/internal/util/apiutil"
)

type deleteClient struct {
	cfg aws.Config
}

func (c deleteClient) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	svc := dynamodb.NewFromConfig(c.cfg)
	return svc.GetItem(ctx, params, optFns...)
}

func (c deleteClient) TransactWriteItems(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	svc := dynamodb.NewFromConfig(c.cfg)
	return svc.TransactWriteItems(ctx, params, optFns...)
}

func (c deleteClient) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	svc := s3.NewFromConfig(c.cfg)
	return svc.DeleteObject(ctx, params, optFns...)
}

func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (apiutil.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(env.Region))
	if err != nil {
		fmt.Printf("unable to load SDK config, %v\n", err)
		return apiutil.NewErrorResponse(http.StatusInternalServerError, "internal error"), nil
	}

	threadID := req.PathParameters["threadID"]
	fmt.Printf("request params: [messagesID] %s\n", threadID)

	if threadID == "" {
		return apiutil.NewErrorResponse(http.StatusBadRequest, "bad request: invalid threadID"), nil
	}

	client := deleteClient{cfg: cfg}
	err = thread.Delete(ctx, client, threadID)
	if err != nil {
		if errors.Is(err, &platform.NotTrashedError{Type: "thread"}) {
			fmt.Printf("dynamodb delete failed: %v\n", err)
			return apiutil.NewErrorResponse(http.StatusBadRequest, "thread not trashed"), nil
		}
		if err == platform.ErrTooManyRequests {
			fmt.Println("too many requests")
			return apiutil.NewErrorResponse(http.StatusTooManyRequests, "too many requests"), nil
		}

		fmt.Printf("dynamodb delete failed: %v\n", err)
		return apiutil.NewErrorResponse(http.StatusInternalServerError, "internal error"), nil
	}

	return apiutil.NewSuccessJSONResponse("{\"status\":\"success\"}"), nil
}

func main() {
	lambda.Start(handler)
}
