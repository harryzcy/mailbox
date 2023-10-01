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
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

	contentID := req.PathParameters["contentID"]
	fmt.Printf("request params: [contentID] %s\n", contentID)
	var disposition string
	if strings.Contains(contentID, "attachments") {
		disposition = "attachment"
	} else {
		disposition = "inline"
	}
	fmt.Printf("request params: [disposition] %s\n", disposition)

	result, err := email.GetContent(ctx, s3.NewFromConfig(cfg), messageID, disposition, contentID)
	if err != nil {
		if err == api.ErrNotFound {
			fmt.Println("not found")
			return apiutil.NewErrorResponse(http.StatusNotFound, "not found"), nil
		}
		if err == api.ErrTooManyRequests {
			fmt.Println("too many requests")
			return apiutil.NewErrorResponse(http.StatusTooManyRequests, "too many requests"), nil
		}
		fmt.Printf("dynamodb get failed: %v\n", err)
		return apiutil.NewErrorResponse(http.StatusInternalServerError, "internal error"), nil
	}

	fmt.Println("invoke successful")
	return apiutil.NewBinaryResponse(
		http.StatusOK, result.Content, result.ContentType,
		disposition, result.Filename,
	), nil
}

func main() {
	lambda.Start(handler)
}
