package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/harryzcy/mailbox/internal/email"
	"github.com/harryzcy/mailbox/internal/util/apiutil"
)

// AWS Region
var region = os.Getenv("REGION")

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (apiutil.Response, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		fmt.Printf("unable to load SDK config, %v\n", err)
		return apiutil.NewErrorResponse(400, "internal error"), nil
	}

	messageID := req.PathParameters["messageID"]
	fmt.Printf("request params: [messagesID] %s\n", messageID)

	if messageID == "" {
		return apiutil.NewErrorResponse(400, "bad request: invalid messageID"), nil
	}

	result, err := email.Get(cfg, messageID)
	if err != nil {
		if err == email.ErrNotFound {
			fmt.Println("email not found")
			return apiutil.NewErrorResponse(404, "email not found"), nil
		}
		fmt.Printf("dynamodb get failed: %v\n", err)
		return apiutil.NewErrorResponse(400, "internal error"), nil
	}

	body, err := json.Marshal(result)
	if err != nil {
		fmt.Printf("marshal failed: %v\n", err)
		return apiutil.NewErrorResponse(400, "internal error"), nil
	}
	fmt.Println("invoke successful")
	return apiutil.NewSuccessJSONResponse(string(body)), nil
}

func main() {
	lambda.Start(handler)
}
