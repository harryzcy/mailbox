package email

import (
	"context"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/util/format"
	"github.com/stretchr/testify/assert"
)

func TestGetThread(t *testing.T) {
	tableName = "table-for-get-thread"
	tests := []struct {
		client      func(t *testing.T) GetItemAPI
		messageID   string
		expected    *Thread
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
							"MessageID":     params.Key["MessageID"],
							"TypeYearMonth": &types.AttributeValueMemberS{Value: "thread#2023-02"},
							"Subject":       &types.AttributeValueMemberS{Value: "subject"},
							"EmailIDs": &types.AttributeValueMemberL{
								Value: []types.AttributeValue{
									&types.AttributeValueMemberS{Value: "id-1"},
									&types.AttributeValueMemberS{Value: "id-2"},
								},
							},
							"TimeUpdated": &types.AttributeValueMemberS{Value: "2022-03-12T01:01:01Z"},
						},
					}, nil
				})
			},
			messageID: "exampleMessageID",
			expected: &Thread{
				MessageID:   "exampleMessageID",
				Type:        "thread",
				Subject:     "subject",
				EmailIDs:    []string{"id-1", "id-2"},
				TimeUpdated: "2022-03-12T01:01:01Z",
			},
		},
		{
			client: func(t *testing.T) GetItemAPI {
				return mockGetItemAPI(func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
					return &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"MessageID":     params.Key["MessageID"],
							"TypeYearMonth": &types.AttributeValueMemberS{Value: "inbox#2023-02"},
						},
					}, nil
				})
			},
			messageID:   "exampleMessageID",
			expected:    nil,
			expectedErr: ErrNotFound,
		},
		{
			client: func(t *testing.T) GetItemAPI {
				return mockGetItemAPI(func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
					return &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"MessageID":     params.Key["MessageID"],
							"TypeYearMonth": &types.AttributeValueMemberS{Value: "invalid#2023-02"},
						},
					}, nil
				})
			},
			messageID:   "exampleMessageID",
			expected:    nil,
			expectedErr: format.ErrInvalidEmailType,
		},
		{
			client: func(t *testing.T) GetItemAPI {
				return mockGetItemAPI(func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
					return &dynamodb.GetItemOutput{
						Item: nil,
					}, nil
				})
			},
			messageID:   "exampleMessageID",
			expected:    nil,
			expectedErr: ErrNotFound,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()
			result, err := GetThread(ctx, test.client(t), test.messageID)
			assert.Equal(t, test.expected, result)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}

type mockGetThreadWithEmailsAPI struct {
	mockGetItem      mockGetItemAPI
	mockBatchGetItem func(ctx context.Context, params *dynamodb.BatchGetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error)
}

func (m mockGetThreadWithEmailsAPI) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return m.mockGetItem(ctx, params, optFns...)
}

func (m mockGetThreadWithEmailsAPI) BatchGetItem(ctx context.Context, params *dynamodb.BatchGetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error) {
	return m.mockBatchGetItem(ctx, params, optFns...)
}

func TestGetThreadWithEmails(t *testing.T) {
	tableName = "table-for-get-thread-with-emails"
	tests := []struct {
		client      func(t *testing.T) GetThreadWithEmailsAPI
		messageID   string
		expected    *Thread
		expectedErr error
	}{
		{
			client: func(t *testing.T) GetThreadWithEmailsAPI {
				return mockGetThreadWithEmailsAPI{
					mockGetItem: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
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
							Item: map[string]dynamodbtypes.AttributeValue{
								"MessageID":     params.Key["MessageID"],
								"TypeYearMonth": &types.AttributeValueMemberS{Value: "thread#2023-02"},
								"Subject":       &types.AttributeValueMemberS{Value: "subject"},
								"EmailIDs": &types.AttributeValueMemberL{
									Value: []types.AttributeValue{
										&types.AttributeValueMemberS{Value: "id-1"},
										&types.AttributeValueMemberS{Value: "id-2"},
									},
								},
								"TimeUpdated": &types.AttributeValueMemberS{Value: "2023-02-18T01:01:01Z"},
							},
						}, nil
					},
					mockBatchGetItem: func(ctx context.Context, params *dynamodb.BatchGetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error) {
						t.Helper()
						assert.NotNil(t, params.RequestItems)
						assert.Len(t, params.RequestItems, 1)
						assert.Len(t, params.RequestItems[tableName].Keys, 2)
						assert.Equal(t,
							params.RequestItems[tableName].Keys[0]["MessageID"].(*types.AttributeValueMemberS).Value,
							"id-1",
						)
						assert.Equal(t,
							params.RequestItems[tableName].Keys[1]["MessageID"].(*types.AttributeValueMemberS).Value,
							"id-2",
						)

						return &dynamodb.BatchGetItemOutput{
							Responses: map[string][]map[string]dynamodbtypes.AttributeValue{
								tableName: {
									{
										"MessageID":     &types.AttributeValueMemberS{Value: "id-1"},
										"TypeYearMonth": &types.AttributeValueMemberS{Value: "inbox#2023-02"},
										"DateTime":      &types.AttributeValueMemberS{Value: "18-01:01:01"},
										"Subject":       &types.AttributeValueMemberS{Value: "subject"},
										"From":          &types.AttributeValueMemberSS{Value: []string{"example@example.com"}},
										"To":            &types.AttributeValueMemberSS{Value: []string{"example@example.com"}},
									},
									{
										"MessageID":     &types.AttributeValueMemberS{Value: "id-2"},
										"TypeYearMonth": &types.AttributeValueMemberS{Value: "inbox#2023-02"},
										"DateTime":      &types.AttributeValueMemberS{Value: "18-01:01:01"},
										"Subject":       &types.AttributeValueMemberS{Value: "subject"},
										"From":          &types.AttributeValueMemberSS{Value: []string{"example@example.com"}},
										"To":            &types.AttributeValueMemberSS{Value: []string{"example@example.com"}},
									},
								},
							},
						}, nil
					},
				}
			},
			messageID: "exampleMessageID",
			expected: &Thread{
				MessageID:   "exampleMessageID",
				Type:        "thread",
				Subject:     "subject",
				EmailIDs:    []string{"id-1", "id-2"},
				TimeUpdated: "2023-02-18T01:01:01Z",
				Emails: []GetResult{
					{
						MessageID:    "id-1",
						Type:         "inbox",
						Subject:      "subject",
						From:         []string{"example@example.com"},
						To:           []string{"example@example.com"},
						TimeReceived: "2023-02-18T01:01:01Z",
						Unread:       aws.Bool(false),
					},
					{
						MessageID:    "id-2",
						Type:         "inbox",
						Subject:      "subject",
						From:         []string{"example@example.com"},
						To:           []string{"example@example.com"},
						TimeReceived: "2023-02-18T01:01:01Z",
						Unread:       aws.Bool(false),
					},
				},
			},
		},
		{
			client: func(t *testing.T) GetThreadWithEmailsAPI {
				return mockGetThreadWithEmailsAPI{
					mockGetItem: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
						return &dynamodb.GetItemOutput{}, nil
					},
				}
			},
			messageID:   "exampleMessageID",
			expected:    nil,
			expectedErr: ErrNotFound,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()
			result, err := GetThreadWithEmails(ctx, test.client(t), test.messageID)
			assert.Equal(t, test.expected, result)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
