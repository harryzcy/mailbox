package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/harryzcy/mailbox/internal/email"
)

// Response is returned from lambda proxy integration
type Response events.APIGatewayProxyResponse

// AWS Region
var region = os.Getenv("REGION")

// Handler ...
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (Response, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	year, err := strconv.Atoi(req.QueryStringParameters["year"])
	if err != nil {
		fmt.Printf("invalid year: %v\n", year)
		return Response{}, errors.New("invalid input")
	}
	month, err := strconv.Atoi(req.QueryStringParameters["month"])
	if err != nil {
		fmt.Printf("invalid month: %v\n", month)
		return Response{}, errors.New("invalid input")
	}

	result, err := email.List(cfg, year, month)
	fmt.Printf("err: %+v\n", err)
	fmt.Printf("result: %+v\n", result)

	body, err := json.Marshal(result)
	if err != nil {
		fmt.Printf("marshal failed: %v\n", err)
		return Response{}, errors.New("marshal failed")
	}
	resp := Response{
		StatusCode:      200,
		IsBase64Encoded: false,
		Body:            string(body),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
	return resp, nil
}

func main() {
	lambda.Start(Handler)
}
