package hook

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqsTypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/platform"
)

// sqsEnabled returns true if SQS is enabled
func sqsEnabled() bool {
	return env.QueueName != ""
}

// SendSQS sends an email receipt to SQS, if SQS is enabled.
// Otherwise, it does nothing.
func SendSQS(ctx context.Context, api platform.SQSSendMessageAPI, input EmailReceipt) error {
	if !sqsEnabled() {
		return nil
	}

	fmt.Printf("Sending email receipt (MessageID: %s)\n", input.MessageID)
	return sendSQSEmailNotification(ctx, api, Hook{
		Event:     EventEmail,
		Action:    ActionReceived,
		Timestamp: input.Timestamp,
		Email: Email{
			ID: input.MessageID,
		},
	})
}

// sendSQSEmailNotification notifies about a change of state of an email, categorized by event.
func sendSQSEmailNotification(ctx context.Context, api platform.SQSSendMessageAPI, input Hook) error {
	result, err := api.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: &env.QueueName,
	})
	if err != nil {
		fmt.Println("Failed to get queue url")
		return err
	}

	body, err := json.Marshal(input)
	if err != nil {
		fmt.Println("Failed to marshal input")
		return err
	}

	resp, err := api.SendMessage(ctx, &sqs.SendMessageInput{
		MessageAttributes: map[string]sqsTypes.MessageAttributeValue{
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
