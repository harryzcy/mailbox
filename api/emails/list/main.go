package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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

	emailType := req.QueryStringParameters["type"]
	year := req.QueryStringParameters["year"]
	month := req.QueryStringParameters["month"]
	order := req.QueryStringParameters["order"]
	showTrash := req.QueryStringParameters["showTrash"]
	pageSizeStr := req.QueryStringParameters["pageSize"]
	nextCursor := req.QueryStringParameters["nextCursor"]

	pageSize := email.DefaultPageSize
	if pageSizeStr != "" {
		var size int64
		size, err = strconv.ParseInt(pageSizeStr, 10, 32)
		if err != nil {
			return apiutil.NewErrorResponse(http.StatusBadRequest, "invalid input"), nil
		}
		pageSize = int32(size) // nolint:gosec
	}

	cursor := &email.Cursor{}
	err = cursor.BindString(nextCursor)
	if err != nil {
		return apiutil.NewErrorResponse(http.StatusBadRequest, "invalid input"), nil
	}

	fmt.Printf("request query: type: %s, year: %s, month: %s, order: %s, pageSize: %s, nextCursor: %s\n",
		emailType, year, month, order, pageSizeStr, nextCursor)

	result, err := email.List(ctx, dynamodb.NewFromConfig(cfg), email.ListInput{
		Type:       emailType,
		Year:       year,
		Month:      month,
		Order:      order,
		ShowTrash:  showTrash,
		PageSize:   pageSize,
		NextCursor: cursor,
	})
	if err != nil {
		if err == platform.ErrInvalidInput {
			return apiutil.NewErrorResponse(http.StatusBadRequest, "invalid input"), nil
		}
		if err == platform.ErrTooManyRequests {
			fmt.Println("too many requests")
			return apiutil.NewErrorResponse(http.StatusTooManyRequests, "too many requests"), nil
		}
		fmt.Printf("email list failed: %v\n", err)
		return apiutil.NewErrorResponse(http.StatusInternalServerError, "internal error"), nil
	}

	body, err := json.Marshal(result)
	if err != nil {
		fmt.Printf("marshal failed: %v\n", err)
		return apiutil.NewErrorResponse(http.StatusInternalServerError, "internal error"), nil
	}
	return apiutil.NewSuccessJSONResponse(string(body)), nil
}

func main() {
	lambda.Start(handler)
}
