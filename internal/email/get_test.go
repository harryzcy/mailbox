package email

import (
	"context"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/platform"
	"github.com/stretchr/testify/assert"
)

type mockGetItemAPI func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)

func (m mockGetItemAPI) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return m(ctx, params, optFns...)
}

func TestGet(t *testing.T) {
	env.TableName = "table-for-get"
	tests := []struct {
		client      func(t *testing.T) platform.GetItemAPI
		messageID   string
		expected    *GetResult
		expectedErr error
	}{
		{
			client: func(t *testing.T) platform.GetItemAPI {
				t.Helper()
				return mockGetItemAPI(func(_ context.Context, params *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
					assert.NotNil(t, params.TableName)
					assert.Equal(t, env.TableName, *params.TableName)

					assert.Len(t, params.Key, 1)
					assert.IsType(t, params.Key["MessageID"], &dynamodbTypes.AttributeValueMemberS{})
					assert.Equal(t,
						params.Key["MessageID"].(*dynamodbTypes.AttributeValueMemberS).Value,
						"exampleMessageID",
					)

					return &dynamodb.GetItemOutput{
						Item: map[string]dynamodbTypes.AttributeValue{
							"MessageID":     &dynamodbTypes.AttributeValueMemberS{Value: "exampleMessageID"},
							"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: "inbox#2022-03"},
							"DateTime":      &dynamodbTypes.AttributeValueMemberS{Value: "12-01:01:01"},
							"Subject":       &dynamodbTypes.AttributeValueMemberS{Value: "subject"},
							"DateSent":      &dynamodbTypes.AttributeValueMemberS{Value: "2022-03-12T01:01:01Z"},
							"Source":        &dynamodbTypes.AttributeValueMemberS{Value: "example@example.com"},
							"Destination":   &dynamodbTypes.AttributeValueMemberSS{Value: []string{"example@example.com"}},
							"From":          &dynamodbTypes.AttributeValueMemberSS{Value: []string{"example@example.com"}},
							"To":            &dynamodbTypes.AttributeValueMemberSS{Value: []string{"example@example.com"}},
							"ReturnPath":    &dynamodbTypes.AttributeValueMemberS{Value: "example@example.com"},
							"Text":          &dynamodbTypes.AttributeValueMemberS{Value: "text"},
							"HTML":          &dynamodbTypes.AttributeValueMemberS{Value: "html"},
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
				Unread:       aws.Bool(false),
			},
		},
		{
			client: func(t *testing.T) platform.GetItemAPI {
				t.Helper()
				return mockGetItemAPI(func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
					return &dynamodb.GetItemOutput{
						Item: map[string]dynamodbTypes.AttributeValue{
							"MessageID":     &dynamodbTypes.AttributeValueMemberS{Value: "exampleMessageID"},
							"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: "draft#2022-03"},
							"DateTime":      &dynamodbTypes.AttributeValueMemberS{Value: "12-01:01:01"},
							"Subject":       &dynamodbTypes.AttributeValueMemberS{Value: "subject"},
							"From":          &dynamodbTypes.AttributeValueMemberSS{Value: []string{"example@example.com"}},
							"To":            &dynamodbTypes.AttributeValueMemberSS{Value: []string{"example@example.com"}},
							"Cc":            &dynamodbTypes.AttributeValueMemberSS{Value: []string{"example@example.com"}},
							"Bcc":           &dynamodbTypes.AttributeValueMemberSS{Value: []string{"example@example.com"}},
							"ReplyTo":       &dynamodbTypes.AttributeValueMemberSS{Value: []string{"example@example.com"}},
							"Text":          &dynamodbTypes.AttributeValueMemberS{Value: "text"},
							"HTML":          &dynamodbTypes.AttributeValueMemberS{Value: "html"},
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
				Unread:      nil,
			},
		},
		{
			client: func(t *testing.T) platform.GetItemAPI {
				t.Helper()
				return mockGetItemAPI(func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
					return &dynamodb.GetItemOutput{}, platform.ErrNotFound
				})
			},
			expectedErr: platform.ErrNotFound,
		},
		{
			client: func(t *testing.T) platform.GetItemAPI {
				t.Helper()
				return mockGetItemAPI(func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
					return &dynamodb.GetItemOutput{
						Item: map[string]dynamodbTypes.AttributeValue{},
					}, nil
				})
			},
			expectedErr: platform.ErrNotFound,
		},
		{
			client: func(t *testing.T) platform.GetItemAPI {
				t.Helper()
				return mockGetItemAPI(func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
					return &dynamodb.GetItemOutput{
						Item: map[string]dynamodbTypes.AttributeValue{},
					}, &dynamodbTypes.ProvisionedThroughputExceededException{}
				})
			},
			expectedErr: platform.ErrTooManyRequests,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Helper()
			ctx := context.TODO()
			result, err := Get(ctx, test.client(t), test.messageID)
			assert.Equal(t, test.expected, result)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}

type mockGetEmailAPI struct {
	mockGetItem    mockGetItemAPI
	mockUpdateItem mockUpdateItemAPI
}

func (m mockGetEmailAPI) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return m.mockGetItem(ctx, params, optFns...)
}

func (m mockGetEmailAPI) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	return m.mockUpdateItem(ctx, params, optFns...)
}

func TestGetAndRead(t *testing.T) {
	env.TableName = "table-for-get-and-read"
	tests := []struct {
		item             map[string]dynamodbTypes.AttributeValue
		expectedUnread   *bool
		expectMarkedRead bool
	}{
		{
			// unread inbox email is marked as read
			item: map[string]dynamodbTypes.AttributeValue{
				"MessageID":     &dynamodbTypes.AttributeValueMemberS{Value: "exampleMessageID"},
				"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: "inbox#2022-03"},
				"DateTime":      &dynamodbTypes.AttributeValueMemberS{Value: "12-01:01:01"},
				"Unread":        &dynamodbTypes.AttributeValueMemberBOOL{Value: true},
			},
			expectedUnread:   aws.Bool(true),
			expectMarkedRead: true,
		},
		{
			// inbox email that's already read is left alone
			item: map[string]dynamodbTypes.AttributeValue{
				"MessageID":     &dynamodbTypes.AttributeValueMemberS{Value: "exampleMessageID"},
				"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: "inbox#2022-03"},
				"DateTime":      &dynamodbTypes.AttributeValueMemberS{Value: "12-01:01:01"},
			},
			expectedUnread: aws.Bool(false),
		},
		{
			// draft email has no read state
			item: map[string]dynamodbTypes.AttributeValue{
				"MessageID":     &dynamodbTypes.AttributeValueMemberS{Value: "exampleMessageID"},
				"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: "draft#2022-03"},
				"DateTime":      &dynamodbTypes.AttributeValueMemberS{Value: "12-01:01:01"},
			},
			expectedUnread: nil,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			markedRead := false
			client := mockGetEmailAPI{
				mockGetItem: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
					return &dynamodb.GetItemOutput{Item: test.item}, nil
				},
				mockUpdateItem: func(_ context.Context, params *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
					markedRead = true
					assert.Equal(t, "REMOVE Unread", *params.UpdateExpression)
					return &dynamodb.UpdateItemOutput{}, nil
				},
			}

			result, err := GetAndRead(context.TODO(), client, "exampleMessageID")
			assert.NoError(t, err)
			assert.Equal(t, test.expectedUnread, result.Unread)
			assert.Equal(t, test.expectMarkedRead, markedRead)
		})
	}
}

// Get, which serves requests that opt out of marking the email as read,
// only needs the read side of the API and thus can't change the read state.
func TestGetDoesNotMarkRead(t *testing.T) {
	env.TableName = "table-for-get-without-read"
	client := mockGetEmailAPI{
		mockGetItem: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]dynamodbTypes.AttributeValue{
					"MessageID":     &dynamodbTypes.AttributeValueMemberS{Value: "exampleMessageID"},
					"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: "inbox#2022-03"},
					"DateTime":      &dynamodbTypes.AttributeValueMemberS{Value: "12-01:01:01"},
					"Unread":        &dynamodbTypes.AttributeValueMemberBOOL{Value: true},
				},
			}, nil
		},
		mockUpdateItem: func(_ context.Context, _ *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			t.Fatal("read state should not be changed")
			return nil, nil
		},
	}

	result, err := Get(context.TODO(), client, "exampleMessageID")
	assert.NoError(t, err)
	assert.Equal(t, aws.Bool(true), result.Unread)
}
