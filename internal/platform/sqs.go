package platform

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// SQSSendMessageAPI defines set of API required by SendEmailReceipt and SendEmailNotification functions
type SQSSendMessageAPI interface {
	//revive:disable:var-naming
	GetQueueUrl(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error)
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}
