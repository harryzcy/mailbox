package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/harryzcy/mailbox/internal/email"
	"github.com/harryzcy/mailbox/internal/util/apiutil"
)

// AWS Region
var region = os.Getenv("REGION")

func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (apiutil.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	fmt.Println("request received")

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
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

	result, err := email.Create(ctx, dynamodb.NewFromConfig(cfg), input)
	if err != nil {
		if err == email.ErrInvalidInput {
			return apiutil.NewErrorResponse(400, "invalid input"), nil
		}
		fmt.Printf("email create failed: %v\n", err)
		return apiutil.NewErrorResponse(400, "internal error"), nil
	}

	body, err := json.Marshal(result)
	if err != nil {
		fmt.Printf("marshal failed: %v\n", err)
		return apiutil.NewErrorResponse(400, "internal error"), nil
	}
	return apiutil.NewSuccessJSONResponse(string(body)), nil
}

func main() {
	lambda.Start(handler)
}
