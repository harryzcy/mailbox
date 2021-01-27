package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/harryzcy/mailbox/internal/email"
	"github.com/harryzcy/mailbox/internal/util"
)

// AWS Region
var region = os.Getenv("REGION")

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (util.Response, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		fmt.Printf("unable to load SDK config, %v\n", err)
		return util.NewErrorResponse(400, "invalid input"), nil
	}

	year, err := strconv.Atoi(req.QueryStringParameters["year"])
	if err != nil {
		fmt.Printf("invalid year: %v\n", year)
		return util.NewErrorResponse(400, "invalid input"), nil
	}
	month, err := strconv.Atoi(req.QueryStringParameters["month"])
	if err != nil {
		fmt.Printf("invalid month: %v\n", month)
		return util.NewErrorResponse(400, "invalid input"), nil
	}

	result, err := email.List(cfg, year, month)
	fmt.Printf("err: %+v\n", err)
	fmt.Printf("result: %+v\n", result)

	body, err := json.Marshal(result)
	if err != nil {
		fmt.Printf("marshal failed: %v\n", err)
		return util.NewErrorResponse(400, "internal error"), nil
	}
	return util.NewSuccessJSONResponse(string(body)), nil
}

func main() {
	lambda.Start(handler)
}
