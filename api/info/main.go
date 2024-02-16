package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/harryzcy/mailbox/internal/util/apiutil"
)

var (
	version   = "dev"
	commit    = "n/a"
	buildDate = "n/a"
)

func handler(_ context.Context, req events.APIGatewayV2HTTPRequest) (apiutil.Response, error) {
	body, err := json.Marshal(map[string]string{
		"version": version,
		"commit":  commit,
		"build":   buildDate,
	})
	if err != nil {
		fmt.Printf("marshal failed: %v\n", err)
		return apiutil.NewErrorResponse(http.StatusInternalServerError, "internal error"), nil
	}
	return apiutil.NewSuccessJSONResponse(string(body)), nil
}

func main() {
	lambda.Start(handler)
}
