package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/harryzcy/mailbox/internal/api"
	"github.com/harryzcy/mailbox/internal/email"
	"github.com/harryzcy/mailbox/internal/env"
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

	var action string
	switch {
	case strings.HasSuffix(req.RequestContext.HTTP.Path, "/unread"):
		action = "unread"
	case strings.HasSuffix(req.RequestContext.HTTP.Path, "/read"):
		action = "read"
	default:
		return apiutil.NewErrorResponse(http.StatusBadRequest, "bad request: invalid action"), nil
	}

	err = email.Read(ctx, dynamodb.NewFromConfig(cfg), messageID, action)
	if err != nil {
		if err == api.ErrTooManyRequests {
			fmt.Println("too many requests")
			return apiutil.NewErrorResponse(http.StatusTooManyRequests, "too many requests"), nil
		}

		fmt.Printf("dynamodb read failed: %v\n", err)
		return apiutil.NewErrorResponse(http.StatusInternalServerError, "internal error"), nil
	}

	return apiutil.NewSuccessJSONResponse("{\"status\":\"success\"}"), nil
}

func main() {
	lambda.Start(handler)
}
