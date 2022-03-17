package email

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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
		client      func(t *testing.T) PutItemAPI
		input       SaveInput
		expected    *SaveResult
		expectedErr error
	}{
		{
			client: func(t *testing.T) PutItemAPI {
				return mockPutItemAPI(func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
					t.Helper()

					assert.Equal(t, tableName, *params.TableName)

					messageID := params.Item["MessageID"].(*types.AttributeValueMemberS).Value
					assert.Equal(t, "exampleMessageID", messageID)

					return &dynamodb.PutItemOutput{}, nil
				})
			},
			input: SaveInput{
				MessageID: "exampleMessageID",
				Subject:   "subject",
				From:      []string{"example@example.com"},
				To:        []string{"example@example.com"},
				Cc:        []string{"example@example.com"},
				Bcc:       []string{"example@example.com"},
				ReplyTo:   []string{"example@example.com"},
				Text:      "text",
				HTML:      "<p>html</p>",
			},
			expected: &SaveResult{
				TimeIndex: TimeIndex{
					MessageID:   "exampleMessageID",
					Type:        EmailTypeDraft,
					TimeCreated: "2022-03-16T16:55:45Z",
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
					return &dynamodb.PutItemOutput{}, ErrInvalidInput
				})
			},
			expectedErr: ErrInvalidInput,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()

			actual, err := Save(ctx, test.client(t), test.input)

			assert.Equal(t, test.expected, actual)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
