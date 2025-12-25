package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	platform "github.com/harryzcy/mailbox/internal/api"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/thread"
	"github.com/harryzcy/mailbox/internal/util/apiutil"
)

func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (apiutil.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	fmt.Println("request received")

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

	result, err := thread.GetThreadWithEmails(ctx, dynamodb.NewFromConfig(cfg), threadID)
	if err != nil {
		if err == platform.ErrNotFound {
			fmt.Println("thread not found")
			return apiutil.NewErrorResponse(http.StatusNotFound, "thread not found"), nil
		}
		if err == platform.ErrTooManyRequests {
			fmt.Println("too many requests")
			return apiutil.NewErrorResponse(http.StatusTooManyRequests, "too many requests"), nil
		}
		fmt.Printf("dynamodb get failed: %v\n", err)
		return apiutil.NewErrorResponse(http.StatusInternalServerError, "internal error"), nil
	}

	body, err := json.Marshal(result)
	if err != nil {
		fmt.Printf("marshal failed: %v\n", err)
		return apiutil.NewErrorResponse(http.StatusInternalServerError, "internal error"), nil
	}
	fmt.Println("invoke successful")
	return apiutil.NewSuccessJSONResponse(string(body)), nil
}

func main() {
	lambda.Start(handler)
}
