package util

import (
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
)

// Response is returned from lambda proxy integration
type Response events.APIGatewayProxyResponse

// ErrorBody represents an error used in response body
type ErrorBody struct {
	Message string `json:"message"`
}

// NewSuccessJSONResponse returns a successful response
func NewSuccessJSONResponse(body string) Response {
	return Response{
		StatusCode:      200,
		IsBase64Encoded: false,
		Body:            body,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

// NewErrorResponse returns an error response
func NewErrorResponse(code int, message string) Response {
	body, _ := json.Marshal(ErrorBody{
		Message: message,
	})
	return Response{
		StatusCode:      code,
		IsBase64Encoded: false,
		Body:            string(body),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}
