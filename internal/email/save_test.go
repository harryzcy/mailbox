package email

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/util/htmlutil"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type mockPutItemAPI func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)

func (m mockPutItemAPI) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	return m(ctx, params, optFns...)
}

func TestTetUpdatedTime(t *testing.T) {
	assert.NotNil(t, getUpdatedTime())
}

func TestSave(t *testing.T) {
	oldGetUpdatedTime := getUpdatedTime
	getUpdatedTime = func() time.Time { return time.Date(2022, 3, 16, 16, 55, 45, 0, time.UTC) }
	defer func() { getUpdatedTime = oldGetUpdatedTime }()

	tableName = "table-for-save"
	tests := []struct {
		client       func(t *testing.T) PutItemAPI
		input        SaveInput
		generateText func(html string) (string, error)
		expected     *SaveResult
		expectedErr  error
	}{
		{
			client: func(t *testing.T) PutItemAPI {
				return mockPutItemAPI(func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
					t.Helper()

					assert.Equal(t, tableName, *params.TableName)

					messageID := params.Item["MessageID"].(*types.AttributeValueMemberS).Value
					assert.Equal(t, "draft-example", messageID)

					assert.Equal(t, "MessageID = :messageID", *params.ConditionExpression)
					assert.Contains(t, params.ExpressionAttributeValues, ":messageID")
					assert.Equal(t, "draft-example",
						params.ExpressionAttributeValues[":messageID"].(*types.AttributeValueMemberS).Value,
					)

					return &dynamodb.PutItemOutput{}, nil
				})
			},
			input: SaveInput{
				EmailInput: EmailInput{
					MessageID: "draft-example",
					Subject:   "subject",
					From:      []string{"example@example.com"},
					To:        []string{"example@example.com"},
					Cc:        []string{"example@example.com"},
					Bcc:       []string{"example@example.com"},
					ReplyTo:   []string{"example@example.com"},
					Text:      "text",
					HTML:      "<p>html</p>",
				},
				GenerateText: "off",
			},
			expected: &SaveResult{
				TimeIndex: TimeIndex{
					MessageID:   "draft-example",
					Type:        EmailTypeDraft,
					TimeUpdated: "2022-03-16T16:55:45Z",
				},
				Subject: "subject",
				From:    []string{"example@example.com"},
				To:      []string{"example@example.com"},
				Cc:      []string{"example@example.com"},
				Bcc:     []string{"example@example.com"},
				ReplyTo: []string{"example@example.com"},
				Text:    "text",
				HTML:    "<p>html</p>",
			},
		},
		{
			client: func(t *testing.T) PutItemAPI {
				return mockPutItemAPI(func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
					return &dynamodb.PutItemOutput{}, nil
				})
			},
			input: SaveInput{
				EmailInput: EmailInput{
					MessageID: "draft-example",
					Subject:   "subject",
					From:      []string{"example@example.com"},
					To:        []string{"example@example.com"},
					Cc:        []string{"example@example.com"},
					Bcc:       []string{"example@example.com"},
					ReplyTo:   []string{"example@example.com"},
					Text:      "text",
					HTML:      "<p>html</p>",
				},
				GenerateText: "on",
			},
			expected: &SaveResult{
				TimeIndex: TimeIndex{
					MessageID:   "draft-example",
					Type:        EmailTypeDraft,
					TimeUpdated: "2022-03-16T16:55:45Z",
				},
				Subject: "subject",
				From:    []string{"example@example.com"},
				To:      []string{"example@example.com"},
				Cc:      []string{"example@example.com"},
				Bcc:     []string{"example@example.com"},
				ReplyTo: []string{"example@example.com"},
				Text:    "html",
				HTML:    "<p>html</p>",
			},
		},
		{
			client: func(t *testing.T) PutItemAPI {
				return mockPutItemAPI(func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
					return &dynamodb.PutItemOutput{}, nil
				})
			},
			input: SaveInput{
				EmailInput: EmailInput{
					MessageID: "draft-example",
					Subject:   "subject",
					From:      []string{"example@example.com"},
					To:        []string{"example@example.com"},
					Cc:        []string{"example@example.com"},
					Bcc:       []string{"example@example.com"},
					ReplyTo:   []string{"example@example.com"},
					HTML:      "<p>html</p>",
				},
				GenerateText: "auto",
			},
			expected: &SaveResult{
				TimeIndex: TimeIndex{
					MessageID:   "draft-example",
					Type:        EmailTypeDraft,
					TimeUpdated: "2022-03-16T16:55:45Z",
				},
				Subject: "subject",
				From:    []string{"example@example.com"},
				To:      []string{"example@example.com"},
				Cc:      []string{"example@example.com"},
				Bcc:     []string{"example@example.com"},
				ReplyTo: []string{"example@example.com"},
				Text:    "html",
				HTML:    "<p>html</p>",
			},
		},
		{
			client: func(t *testing.T) PutItemAPI {
				return mockPutItemAPI(func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
					return &dynamodb.PutItemOutput{}, nil
				})
			},
			input: SaveInput{
				EmailInput: EmailInput{
					MessageID: "draft-example",
				},
				GenerateText: "on",
			},
			generateText: func(html string) (string, error) {
				return "", ErrInvalidInput
			},
			expectedErr: ErrInvalidInput,
		},
		{
			client: func(t *testing.T) PutItemAPI {
				return mockPutItemAPI(func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
					return &dynamodb.PutItemOutput{}, ErrInvalidInput
				})
			},
			input: SaveInput{
				EmailInput: EmailInput{
					MessageID: "draft-example",
				},
			},
			expectedErr: ErrInvalidInput,
		},
		{
			client: func(t *testing.T) PutItemAPI {
				return mockPutItemAPI(func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
					t.Helper()
					t.Error("this mock shouldn't be reached")
					return &dynamodb.PutItemOutput{}, nil
				})
			},
			expectedErr: ErrEmailIsNotDraft,
		},
		{
			client: func(t *testing.T) PutItemAPI {
				return mockPutItemAPI(func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
					return &dynamodb.PutItemOutput{}, errors.Wrap(&types.ConditionalCheckFailedException{}, "")
				})
			},
			input: SaveInput{
				EmailInput: EmailInput{
					MessageID: "draft-example",
				},
			},
			expectedErr: ErrNotFound,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()

			if test.generateText != nil {
				generateText = test.generateText
			} else {
				generateText = htmlutil.GenerateText
			}

			actual, err := Save(ctx, test.client(t), test.input)

			assert.Equal(t, test.expected, actual)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
