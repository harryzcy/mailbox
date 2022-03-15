package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

var (
	queueName = os.Getenv("SQS_QUEUE")
)

// SQSStorage references all SQS related functions
type SQSStorage interface {
	// SendEmailHandle sends an email receipt to SQS.
	SendEmailReceipt(ctx context.Context, api SQSSendMessageAPI, input EmailReceipt) error
	// SendEmailNotification notifies about the state change of an email, categorized by event.
	SendEmailNotification(ctx context.Context, api SQSSendMessageAPI, input EmailNotification) error
}

type sqsStorage struct{}

// SQS is the default implementation of SQSStorage
var SQS SQSStorage = sqsStorage{}

// SQSSendMessageAPI defines set of API required by SendEmailReceipt and SendEmailNotification functions
type SQSSendMessageAPI interface {
	GetQueueUrl(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error)
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

// EmailReceipt contains information needed for an email receipt
type EmailReceipt struct {
	MessageID string
	Timestamp string
}

// SendEmailHandle sends an email receipt to SQS.
// This function wraps around SendEmailNotification.
func (s sqsStorage) SendEmailReceipt(ctx context.Context, api SQSSendMessageAPI, input EmailReceipt) error {
	fmt.Printf("Sending email receipt (MessageID: %s)\n", input.MessageID)
	return s.SendEmailNotification(ctx, api, EmailNotification{
		Event:     "receive",
		MessageID: input.MessageID,
		Timestamp: input.Timestamp,
	})
}

// EmailNotification contains information needed for an email state change notification
type EmailNotification struct {
	Event     string `json:"event"`
	MessageID string `json:"messageID"`
	Timestamp string `json:"timestamp"`
}

// SendEmailNotification notifies about a change of state of an email, categorized by event.
func (s sqsStorage) SendEmailNotification(ctx context.Context, api SQSSendMessageAPI, input EmailNotification) error {
	result, err := api.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: &queueName,
	})
	if err != nil {
		fmt.Println("Failed to get queue url")
		return err
	}

	body, _ := json.Marshal(input)

	resp, err := api.SendMessage(ctx, &sqs.SendMessageInput{
		MessageAttributes: map[string]types.MessageAttributeValue{
			"Event": {
				DataType:    aws.String("String"),
				StringValue: aws.String(input.Event),
			},
			"Timestamp": {
				DataType:    aws.String("String"),
				StringValue: aws.String(input.Timestamp),
			},
		},
		MessageBody: aws.String(string(body)),
		QueueUrl:    result.QueueUrl,
	})
	if err != nil {
		fmt.Println("Failed to send message to SQS")
		return err
	}

	fmt.Println("Sent message with ID: " + *resp.MessageId)
	return nil
}
