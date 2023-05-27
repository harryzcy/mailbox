package storage

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/stretchr/testify/assert"
)

type mockSQSSendMessageAPI struct {
	mockGetQueueUrl func(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error)
	mockSendMessage func(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

func (m mockSQSSendMessageAPI) GetQueueUrl(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error) {
	return m.mockGetQueueUrl(ctx, params, optFns...)
}

func (m mockSQSSendMessageAPI) SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	return m.mockSendMessage(ctx, params, optFns...)
}

func TestSQSEnabled(t *testing.T) {
	env.QueueName = "test-queue-TestSQSEnabled"
	assert.True(t, SQS.Enabled())

	env.QueueName = ""
	assert.False(t, SQS.Enabled())
}

func TestSQSSendMessageAPI(t *testing.T) {
	env.QueueName = "test-queue-TestSQSSendMessageAPI"
	tests := []struct {
		client      func(t *testing.T) SQSSendMessageAPI
		input       EmailReceipt
		expectedErr error
	}{
		{
			client: func(t *testing.T) SQSSendMessageAPI {
				return mockSQSSendMessageAPI{
					mockGetQueueUrl: func(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error) {
						return &sqs.GetQueueUrlOutput{
							QueueUrl: aws.String("https://queue.url"),
						}, nil
					},
					mockSendMessage: func(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
						return &sqs.SendMessageOutput{
							MessageId: aws.String("MessageId"),
						}, nil
					},
				}
			},
			input: EmailReceipt{
				MessageID: "exampleMessageID",
				Timestamp: "2022-03-12T10:10:10Z",
			},
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()

			err := SQS.SendEmailReceipt(ctx, test.client(t), test.input)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}

func TestSendEmailNotification(t *testing.T) {
	env.QueueName = "test-queue-TestSendEmailNotification"
	tests := []struct {
		client      func(t *testing.T) SQSSendMessageAPI
		input       EmailNotification
		expectedErr error
	}{
		{
			client: func(t *testing.T) SQSSendMessageAPI {
				return mockSQSSendMessageAPI{
					mockGetQueueUrl: func(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error) {
						t.Helper()
						assert.Equal(t, env.QueueName, *params.QueueName)

						return &sqs.GetQueueUrlOutput{
							QueueUrl: aws.String("https://queue.url"),
						}, nil
					},
					mockSendMessage: func(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
						t.Helper()
						assert.Equal(t, "https://queue.url", *params.QueueUrl)

						assert.Len(t, params.MessageAttributes, 2)
						assert.Contains(t, params.MessageAttributes, "Event")
						assert.Contains(t, params.MessageAttributes, "Timestamp")
						assert.Equal(t, types.MessageAttributeValue{
							DataType:    aws.String("String"),
							StringValue: aws.String("received"),
						}, params.MessageAttributes["Event"])
						assert.Equal(t, types.MessageAttributeValue{
							DataType:    aws.String("String"),
							StringValue: aws.String("2022-03-12T10:10:10Z"),
						}, params.MessageAttributes["Timestamp"])

						assert.Contains(t, *params.MessageBody, "\"event\":\"received\"")
						assert.Contains(t, *params.MessageBody, "\"timestamp\":\"2022-03-12T10:10:10Z\"")
						assert.Contains(t, *params.MessageBody, "\"messageID\":\"exampleMessageID\"")

						return &sqs.SendMessageOutput{
							MessageId: aws.String("MessageId"),
						}, nil
					},
				}
			},
			input: EmailNotification{
				Event:     "received",
				MessageID: "exampleMessageID",
				Timestamp: "2022-03-12T10:10:10Z",
			},
		},
		{
			client: func(t *testing.T) SQSSendMessageAPI {
				return mockSQSSendMessageAPI{
					mockGetQueueUrl: func(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error) {
						return &sqs.GetQueueUrlOutput{}, errors.New("some-error")
					},
				}
			},
			expectedErr: errors.New("some-error"),
		},
		{
			client: func(t *testing.T) SQSSendMessageAPI {
				return mockSQSSendMessageAPI{
					mockGetQueueUrl: func(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error) {
						return &sqs.GetQueueUrlOutput{
							QueueUrl: aws.String("https://queue.url"),
						}, nil
					},
					mockSendMessage: func(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
						return &sqs.SendMessageOutput{}, errors.New("some-other-error")
					},
				}
			},
			expectedErr: errors.New("some-other-error"),
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()

			err := SQS.SendEmailNotification(ctx, test.client(t), test.input)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
