package hook

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/harryzcy/mailbox/internal/api"
	"github.com/harryzcy/mailbox/internal/env"
)

// EmailReceipt contains information needed for an email receipt
type EmailReceipt struct {
	MessageID string
	Timestamp string
}

// sqsEnabled returns true if SQS is enabled
func sqsEnabled() bool {
	return env.QueueName != ""
}

// SendEmailHandle sends an email receipt to SQS, if SQS is enabled.
// Otherwise, it does nothing.
func SendSQS(ctx context.Context, api api.SQSSendMessageAPI, input EmailReceipt) error {
	if !sqsEnabled() {
		return nil
	}

	fmt.Printf("Sending email receipt (MessageID: %s)\n", input.MessageID)
	return sendSQSEmailNotification(ctx, api, EmailNotification{
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

// sendSQSEmailNotification notifies about a change of state of an email, categorized by event.
func sendSQSEmailNotification(ctx context.Context, api api.SQSSendMessageAPI, input EmailNotification) error {
	result, err := api.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: &env.QueueName,
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
