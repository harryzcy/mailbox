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
	"github.com/harryzcy/mailbox/internal/email"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/platform"
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

	messageID := req.PathParameters["messageID"]
	fmt.Printf("request params: [messagesID] %s\n", messageID)

	if messageID == "" {
		return apiutil.NewErrorResponse(http.StatusBadRequest, "bad request: invalid messageID"), nil
	}

	err = email.Trash(ctx, dynamodb.NewFromConfig(cfg), messageID)
	if err != nil {
		if errors.Is(err, &platform.AlreadyTrashedError{Type: "email"}) {
			fmt.Printf("dynamodb trash failed: %v\n", err)
			return apiutil.NewErrorResponse(http.StatusBadRequest, "email is already trashed"), nil
		}
		if err == platform.ErrTooManyRequests {
			fmt.Println("too many requests")
			return apiutil.NewErrorResponse(http.StatusTooManyRequests, "too many requests"), nil
		}

		fmt.Printf("dynamodb trash failed: %v\n", err)
		return apiutil.NewErrorResponse(http.StatusInternalServerError, "internal error"), nil
	}

	return apiutil.NewSuccessJSONResponse("{\"status\":\"success\"}"), nil
}

func main() {
	lambda.Start(handler)
}
