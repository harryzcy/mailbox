package email

import (
	"context"
	"strconv"
	"strings"
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

func TestTetCreatedTime(t *testing.T) {
	assert.NotNil(t, getCreatedTime())
}

func TestCreate(t *testing.T) {
	oldGetCreatedTime := getCreatedTime
	getCreatedTime = func() time.Time { return time.Date(2022, 3, 16, 16, 55, 45, 0, time.UTC) }
	defer func() { getCreatedTime = oldGetCreatedTime }()

	tableName = "table-for-create"
	tests := []struct {
		client      func(t *testing.T) PutItemAPI
		input       CreateInput
		expected    *CreateResult
		expectedErr error
	}{
		{
			client: func(t *testing.T) PutItemAPI {
				return mockPutItemAPI(func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
					t.Helper()

					assert.Equal(t, tableName, *params.TableName)

					messageID := params.Item["MessageID"].(*types.AttributeValueMemberS).Value
					assert.Len(t, messageID, 6+32)
					assert.True(t, strings.HasPrefix(messageID, "draft-"))

					return &dynamodb.PutItemOutput{}, nil
				})
			},
			input: CreateInput{
				Subject: "subject",
				From:    []string{"example@example.com"},
				To:      []string{"example@example.com"},
				Cc:      []string{"example@example.com"},
				Bcc:     []string{"example@example.com"},
				ReplyTo: []string{"example@example.com"},
				Text:    "text",
				HTML:    "<p>html</p>",
			},
			expected: &CreateResult{
				DraftIndex: DraftIndex{
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

			actual, err := Create(ctx, test.client(t), test.input)

			if actual != nil && test.expected != nil {
				assert.True(t, strings.HasPrefix(actual.MessageID, "draft-"))
				test.expected.MessageID = actual.MessageID // messageID is randomly generated
			}

			assert.Equal(t, test.expected, actual)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
