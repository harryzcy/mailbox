package email

import (
	"context"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
)

type mockGetItemAPI func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)

func (m mockGetItemAPI) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return m(ctx, params, optFns...)
}

func TestGet(t *testing.T) {
	tableName = "table-for-get"
	tests := []struct {
		client      func(t *testing.T) GetItemAPI
		messageID   string
		expected    *GetResult
		expectedErr error
	}{
		{
			client: func(t *testing.T) GetItemAPI {
				return mockGetItemAPI(func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
					t.Helper()
					assert.NotNil(t, params.TableName)
					assert.Equal(t, tableName, *params.TableName)

					assert.Len(t, params.Key, 1)
					assert.IsType(t, params.Key["MessageID"], &types.AttributeValueMemberS{})
					assert.Equal(t,
						params.Key["MessageID"].(*types.AttributeValueMemberS).Value,
						"exampleMessageID",
					)

					return &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"MessageID":     &types.AttributeValueMemberS{Value: "exampleMessageID"},
							"TypeYearMonth": &types.AttributeValueMemberS{Value: "inbox#2022-03"},
							"DateTime":      &types.AttributeValueMemberS{Value: "12-01:01:01"},
							"Subject":       &types.AttributeValueMemberS{Value: "subject"},
							"DateSent":      &types.AttributeValueMemberS{Value: "2022-03-12T01:01:01Z"},
							"Source":        &types.AttributeValueMemberS{Value: "example@example.com"},
							"Destination":   &types.AttributeValueMemberSS{Value: []string{"example@example.com"}},
							"From":          &types.AttributeValueMemberSS{Value: []string{"example@example.com"}},
							"To":            &types.AttributeValueMemberSS{Value: []string{"example@example.com"}},
							"ReturnPath":    &types.AttributeValueMemberS{Value: "example@example.com"},
							"Text":          &types.AttributeValueMemberS{Value: "text"},
							"HTML":          &types.AttributeValueMemberS{Value: "html"},
						},
					}, nil
				})
			},
			messageID: "exampleMessageID",
			expected: &GetResult{
				MessageID:    "exampleMessageID",
				Type:         "inbox",
				TimeReceived: "2022-03-12T01:01:01Z",
				Subject:      "subject",
				DateSent:     "2022-03-12T01:01:01Z",
				Source:       "example@example.com",
				Destination:  []string{"example@example.com"},
				From:         []string{"example@example.com"},
				To:           []string{"example@example.com"},
				ReturnPath:   "example@example.com",
				Text:         "text",
				HTML:         "html",
			},
		},
		{
			client: func(t *testing.T) GetItemAPI {
				return mockGetItemAPI(func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
					return &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"MessageID":     &types.AttributeValueMemberS{Value: "exampleMessageID"},
							"TypeYearMonth": &types.AttributeValueMemberS{Value: "draft#2022-03"},
							"DateTime":      &types.AttributeValueMemberS{Value: "12-01:01:01"},
							"Subject":       &types.AttributeValueMemberS{Value: "subject"},
							"From":          &types.AttributeValueMemberSS{Value: []string{"example@example.com"}},
							"To":            &types.AttributeValueMemberSS{Value: []string{"example@example.com"}},
							"Cc":            &types.AttributeValueMemberSS{Value: []string{"example@example.com"}},
							"Bcc":           &types.AttributeValueMemberSS{Value: []string{"example@example.com"}},
							"ReplyTo":       &types.AttributeValueMemberSS{Value: []string{"example@example.com"}},
							"Text":          &types.AttributeValueMemberS{Value: "text"},
							"HTML":          &types.AttributeValueMemberS{Value: "html"},
						},
					}, nil
				})
			},
			messageID: "exampleMessageID",
			expected: &GetResult{
				MessageID:   "exampleMessageID",
				Type:        "draft",
				TimeUpdated: "2022-03-12T01:01:01Z",
				Subject:     "subject",
				From:        []string{"example@example.com"},
				To:          []string{"example@example.com"},
				Cc:          []string{"example@example.com"},
				Bcc:         []string{"example@example.com"},
				ReplyTo:     []string{"example@example.com"},
				Text:        "text",
				HTML:        "html",
			},
		},
		{
			client: func(t *testing.T) GetItemAPI {
				return mockGetItemAPI(func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
					return &dynamodb.GetItemOutput{}, ErrNotFound
				})
			},
			expectedErr: ErrNotFound,
		},
		{
			client: func(t *testing.T) GetItemAPI {
				return mockGetItemAPI(func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
					return &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{},
					}, nil
				})
			},
			expectedErr: ErrNotFound,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()
			result, err := Get(ctx, test.client(t), test.messageID)
			assert.Equal(t, test.expected, result)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
