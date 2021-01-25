package main

import (
	"context"
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
		return Response{}, errors.New("invalid input")
	}
	month, err := strconv.Atoi(req.QueryStringParameters["month"])
	if err != nil {
		return Response{}, errors.New("invalid input")
	}

	count, err := email.List(cfg, year, month)
	fmt.Printf("err: %+v\n", err)
	fmt.Printf("count: %+v\n", count)

	resp := Response{
		StatusCode:      200,
		IsBase64Encoded: false,
		Body:            "hello",
		Headers: map[string]string{
			"Content-Type": "text/plain",
		},
	}
	return resp, nil
}

func main() {
	lambda.Start(Handler)
}
