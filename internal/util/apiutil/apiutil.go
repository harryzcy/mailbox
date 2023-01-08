package apiutil

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
)

// Response is returned from lambda proxy integration
type Response events.APIGatewayProxyResponse

// ErrorBody represents an error used in response body
type ErrorBody struct {
	Message string `json:"message"`
}

// NewBinaryResponse returns a binary response.
// If filename is not empty, it will be used as the filename in the Content-Disposition header.
func NewBinaryResponse(code int, content []byte, contentType, disposition, filename string) Response {
	body := base64.StdEncoding.EncodeToString(content)

	contentDisposition := disposition
	if filename != "" {
		contentDisposition += fmt.Sprintf("; filename=\"%s\"", filename)
	}
	return Response{
		StatusCode:      200,
		IsBase64Encoded: true,
		Body:            body,
		Headers: map[string]string{
			"Content-Type":        contentType,
			"Content-Disposition": contentDisposition,
		},
	}
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
