package email

import (
	"context"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go/middleware"
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

func TestGenerateThreadID(t *testing.T) {
	id := generateThreadID()
	assert.NotEmpty(t, id)
	assert.Len(t, id, 36-4) // minus the 4 dashes
	assert.NotContains(t, id, "-")
}

type mockTransactWriteItemAPI func(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)

func (m mockTransactWriteItemAPI) TransactWriteItems(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	return m(ctx, params, optFns...)
}

func TestStoreEmailWithExistingThread(t *testing.T) {
	tableName = "table-for-store-email-with-existing-thread"
	tests := []struct {
		client            func(t *testing.T) TransactWriteItemsAPI
		threadID          string
		email             map[string]dynamodbTypes.AttributeValue
		timeReceived      string
		previousMessageID string
		expectedErr       error
	}{
		{
			client: func(t *testing.T) TransactWriteItemsAPI {
				return mockTransactWriteItemAPI(func(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
					t.Helper()

					for _, item := range params.TransactItems {
						if item.Put != nil {
							assert.Equal(t, tableName, *item.Put.TableName)
							assert.Equal(t, map[string]dynamodbTypes.AttributeValue{
								"MessageID":      &dynamodbtypes.AttributeValueMemberS{Value: "exampleMessageID"},
								"IsThreadLatest": &dynamodbtypes.AttributeValueMemberBOOL{Value: true},
							}, item.Put.Item)
						}
						if item.Update != nil {
							assert.Equal(t, tableName, *item.Update.TableName)
							assert.IsType(t, item.Update.Key["MessageID"], &types.AttributeValueMemberS{})

							if item.Update.Key["MessageID"].(*types.AttributeValueMemberS).Value == "exampleThreadID" {
								assert.Equal(t, "SET #emails = list_append(#emails, :emails), #timeUpdated = :timeUpdated", *item.Update.UpdateExpression)
								assert.Equal(t, map[string]string{
									"#emails":      "EmailIDs",
									"#timeUpdated": "TimeUpdated",
								}, item.Update.ExpressionAttributeNames)
								assert.Equal(t, map[string]types.AttributeValue{
									":emails": &types.AttributeValueMemberL{
										Value: []types.AttributeValue{
											&types.AttributeValueMemberS{Value: "exampleMessageID"},
										},
									},
									":timeUpdated": &types.AttributeValueMemberS{Value: "2023-02-18T01:01:01Z"},
								}, item.Update.ExpressionAttributeValues)
							} else {
								assert.Equal(t, "examplePreviousMessageID", item.Update.Key["MessageID"].(*types.AttributeValueMemberS).Value)
								assert.Equal(t, "REMOVE IsThreadLatest", *item.Update.UpdateExpression)
							}
						}
					}

					return &dynamodb.TransactWriteItemsOutput{
						ResultMetadata: middleware.Metadata{},
					}, nil
				})
			},
			threadID: "exampleThreadID",
			email: map[string]dynamodbtypes.AttributeValue{
				"MessageID": &dynamodbtypes.AttributeValueMemberS{Value: "exampleMessageID"},
			},
			timeReceived:      "2023-02-18T01:01:01Z",
			previousMessageID: "examplePreviousMessageID",
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()
			err := StoreEmailWithExistingThread(ctx, test.client(t), &StoreEmailWithExistingThreadInput{
				ThreadID:          test.threadID,
				Email:             test.email,
				TimeReceived:      test.timeReceived,
				PreviousMessageID: test.previousMessageID,
			})
			assert.Equal(t, test.expectedErr, err)
		})
	}
}

func TestStoreEmailWithNewThread(t *testing.T) {
	tableName = "table-for-store-email-with-existing-thread"
	tests := []struct {
		client          func(t *testing.T) TransactWriteItemsAPI
		threadID        string
		email           map[string]dynamodbTypes.AttributeValue
		CreatingEmailID string
		CreatingSubject string
		CreatingTime    string
		TimeReceived    string
		expectedErr     error
	}{
		{
			client: func(t *testing.T) TransactWriteItemsAPI {
				return mockTransactWriteItemAPI(func(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
					t.Helper()
					for _, item := range params.TransactItems {
						if item.Put != nil {
							assert.Equal(t, tableName, *item.Put.TableName)

							if item.Put.Item["MessageID"].(*dynamodbtypes.AttributeValueMemberS).Value == "exampleThreadID" {
								assert.Equal(t, map[string]dynamodbTypes.AttributeValue{
									"MessageID": &dynamodbtypes.AttributeValueMemberS{Value: "exampleThreadID"},
									"TypeYearMonth": &dynamodbtypes.AttributeValueMemberS{
										Value: "thread#2023-02",
									},
									"EmailIDs": &dynamodbtypes.AttributeValueMemberL{
										Value: []types.AttributeValue{
											&types.AttributeValueMemberS{Value: "exampleCreatingEmailID"},
											&types.AttributeValueMemberS{Value: "exampleMessageID"},
										},
									},
									"TimeUpdated": &dynamodbtypes.AttributeValueMemberS{Value: "2023-02-19T01:01:01Z"},
									"Subject":     &dynamodbtypes.AttributeValueMemberS{Value: "exampleCreatingSubject"},
								}, item.Put.Item)
							} else if item.Put.Item["MessageID"].(*dynamodbtypes.AttributeValueMemberS).Value == "exampleMessageID" {
								assert.Equal(t, map[string]dynamodbTypes.AttributeValue{
									"MessageID":      &dynamodbtypes.AttributeValueMemberS{Value: "exampleMessageID"},
									"IsThreadLatest": &dynamodbtypes.AttributeValueMemberBOOL{Value: true},
								}, item.Put.Item)
							} else {
								assert.Fail(t, "unexpected item", item.Put.Item)
							}
						}
						if item.Update != nil {
							assert.Equal(t, tableName, *item.Update.TableName)
							assert.IsType(t, item.Update.Key["MessageID"], &types.AttributeValueMemberS{})
							assert.Equal(t,
								item.Update.Key["MessageID"].(*types.AttributeValueMemberS).Value,
								"exampleCreatingEmailID",
							)
							assert.Equal(t, "SET #threadID = :threadID", *item.Update.UpdateExpression)
							assert.Equal(t, map[string]string{
								"#threadID": "ThreadID",
							}, item.Update.ExpressionAttributeNames)
							assert.Equal(t, map[string]types.AttributeValue{
								":threadID": &types.AttributeValueMemberS{Value: "exampleThreadID"},
							}, item.Update.ExpressionAttributeValues)
						}
					}

					return &dynamodb.TransactWriteItemsOutput{
						ResultMetadata: middleware.Metadata{},
					}, nil
				})
			},
			threadID: "exampleThreadID",
			email: map[string]dynamodbtypes.AttributeValue{
				"MessageID": &dynamodbtypes.AttributeValueMemberS{Value: "exampleMessageID"},
			},
			CreatingEmailID: "exampleCreatingEmailID",
			CreatingSubject: "exampleCreatingSubject",
			CreatingTime:    "2023-02-18T01:01:01Z",
			TimeReceived:    "2023-02-19T01:01:01Z",
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()
			err := StoreEmailWithNewThread(ctx, test.client(t), &StoreEmailWithNewThreadInput{
				ThreadID:        test.threadID,
				Email:           test.email,
				CreatingEmailID: test.CreatingEmailID,
				CreatingSubject: test.CreatingSubject,
				CreatingTime:    test.CreatingTime,
				TimeReceived:    test.TimeReceived,
			})
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
