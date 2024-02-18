package hook

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/harryzcy/mailbox/internal/api"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/stretchr/testify/assert"
)

type mockSQSSendMessageAPI struct {
	mockGetQueueURL func(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error)
	mockSendMessage func(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

//revive:disable:var-naming
func (m mockSQSSendMessageAPI) GetQueueUrl(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error) {
	return m.mockGetQueueURL(ctx, params, optFns...)
}

func (m mockSQSSendMessageAPI) SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	return m.mockSendMessage(ctx, params, optFns...)
}

var _ api.SQSSendMessageAPI = mockSQSSendMessageAPI{}

func TestSQSEnabled(t *testing.T) {
	env.QueueName = "test-queue-TestSQSEnabled"
	assert.True(t, sqsEnabled())

	env.QueueName = ""
	assert.False(t, sqsEnabled())
}

func TestSendSQS(t *testing.T) {
	env.QueueName = "test-queue-TestSQSSendMessageAPI"
	tests := []struct {
		client      func(t *testing.T) api.SQSSendMessageAPI
		input       EmailReceipt
		expectedErr error
	}{
		{
			client: func(t *testing.T) api.SQSSendMessageAPI {
				t.Helper()
				return mockSQSSendMessageAPI{
					mockGetQueueURL: func(_ context.Context, _ *sqs.GetQueueUrlInput, _ ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error) {
						return &sqs.GetQueueUrlOutput{
							QueueUrl: aws.String("https://queue.url"),
						}, nil
					},
					mockSendMessage: func(_ context.Context, _ *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
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

			err := SendSQS(ctx, test.client(t), test.input)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}

func TestSendSQS_NoOp(t *testing.T) {
	env.QueueName = ""
	err := SendSQS(context.Background(), nil, EmailReceipt{})
	assert.Nil(t, err)
}

func TestSendSQSEmailNotification(t *testing.T) {
	env.QueueName = "test-queue-TestSendEmailNotification"
	tests := []struct {
		client      func(t *testing.T) api.SQSSendMessageAPI
		input       Hook
		expectedErr error
	}{
		{
			client: func(t *testing.T) api.SQSSendMessageAPI {
				return mockSQSSendMessageAPI{
					mockGetQueueURL: func(_ context.Context, params *sqs.GetQueueUrlInput, _ ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error) {
						t.Helper()
						assert.Equal(t, env.QueueName, *params.QueueName)

						return &sqs.GetQueueUrlOutput{
							QueueUrl: aws.String("https://queue.url"),
						}, nil
					},
					mockSendMessage: func(_ context.Context, params *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
						t.Helper()
						assert.Equal(t, "https://queue.url", *params.QueueUrl)

						assert.Len(t, params.MessageAttributes, 2)
						assert.Contains(t, params.MessageAttributes, "Event")
						assert.Contains(t, params.MessageAttributes, "Timestamp")
						assert.Equal(t, types.MessageAttributeValue{
							DataType:    aws.String("String"),
							StringValue: aws.String("email"),
						}, params.MessageAttributes["Event"])
						assert.Equal(t, types.MessageAttributeValue{
							DataType:    aws.String("String"),
							StringValue: aws.String("2022-03-12T10:10:10Z"),
						}, params.MessageAttributes["Timestamp"])

						assert.Contains(t, *params.MessageBody, "\"event\":\"email\"")
						assert.Contains(t, *params.MessageBody, "\"action\":\"received\"")
						assert.Contains(t, *params.MessageBody, "\"timestamp\":\"2022-03-12T10:10:10Z\"")
						assert.Contains(t, *params.MessageBody, "\"id\":\"exampleMessageID\"")

						return &sqs.SendMessageOutput{
							MessageId: aws.String("MessageId"),
						}, nil
					},
				}
			},
			input: Hook{
				Event:     EventEmail,
				Action:    ActionReceived,
				Timestamp: "2022-03-12T10:10:10Z",
				Email: Email{
					ID: "exampleMessageID",
				},
			},
		},
		{
			client: func(t *testing.T) api.SQSSendMessageAPI {
				t.Helper()
				return mockSQSSendMessageAPI{
					mockGetQueueURL: func(_ context.Context, _ *sqs.GetQueueUrlInput, _ ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error) {
						return &sqs.GetQueueUrlOutput{}, errors.New("some-error")
					},
				}
			},
			expectedErr: errors.New("some-error"),
		},
		{
			client: func(t *testing.T) api.SQSSendMessageAPI {
				t.Helper()
				return mockSQSSendMessageAPI{
					mockGetQueueURL: func(_ context.Context, _ *sqs.GetQueueUrlInput, _ ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error) {
						return &sqs.GetQueueUrlOutput{
							QueueUrl: aws.String("https://queue.url"),
						}, nil
					},
					mockSendMessage: func(_ context.Context, _ *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
						return &sqs.SendMessageOutput{}, errors.New("some-other-error")
					},
				}
			},
			expectedErr: errors.New("some-other-error"),
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.Background()
			err := sendSQSEmailNotification(ctx, test.client(t), test.input)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
