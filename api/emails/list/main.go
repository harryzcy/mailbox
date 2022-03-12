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
		return apiutil.NewErrorResponse(400, "invalid input"), nil
	}

	year := req.QueryStringParameters["year"]
	month := req.QueryStringParameters["month"]

	fmt.Printf("request received: year: %s, month %s", year, month)

	result, err := email.List(cfg, year, month)
	if err != nil {
		if err == email.ErrInvalidInput {
			return apiutil.NewErrorResponse(400, "invalid input"), nil
		}
		fmt.Printf("email list failed: %v\n", err)
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
