package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/platform"
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

	err = thread.Untrash(ctx, dynamodb.NewFromConfig(cfg), threadID)
	if err != nil {
		if errors.Is(err, &platform.NotTrashedError{Type: "thread"}) {
			fmt.Printf("dynamodb untrash failed: %v\n", err)
			return apiutil.NewErrorResponse(http.StatusBadRequest, "thread already not trashed"), nil
		}
		if err == platform.ErrTooManyRequests {
			fmt.Println("too many requests")
			return apiutil.NewErrorResponse(http.StatusTooManyRequests, "too many requests"), nil
		}

		fmt.Printf("dynamodb untrash failed: %v\n", err)
		return apiutil.NewErrorResponse(http.StatusInternalServerError, "internal error"), nil
	}

	return apiutil.NewSuccessJSONResponse("{\"status\":\"success\"}"), nil
}

func main() {
	lambda.Start(handler)
}
