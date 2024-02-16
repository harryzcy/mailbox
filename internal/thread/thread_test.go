package thread

import (
	"context"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/harryzcy/mailbox/internal/api"
	"github.com/harryzcy/mailbox/internal/email"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/util/format"
	"github.com/harryzcy/mailbox/internal/util/idutil"
	"github.com/harryzcy/mailbox/internal/util/mockutil"
	"github.com/stretchr/testify/assert"
)

func TestGetThread(t *testing.T) {
	env.TableName = "table-for-get-thread"
	tests := []struct {
		client      func(t *testing.T) api.GetItemAPI
		messageID   string
		expected    *Thread
		expectedErr error
	}{
		{
			client: func(t *testing.T) api.GetItemAPI {
				return mockutil.MockGetItemAPI(func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
					t.Helper()
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
							"MessageID":     params.Key["MessageID"],
							"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: "thread#2023-02"},
							"Subject":       &dynamodbTypes.AttributeValueMemberS{Value: "subject"},
							"EmailIDs": &dynamodbTypes.AttributeValueMemberL{
								Value: []dynamodbTypes.AttributeValue{
									&dynamodbTypes.AttributeValueMemberS{Value: "id-1"},
									&dynamodbTypes.AttributeValueMemberS{Value: "id-2"},
								},
							},
							"TimeUpdated": &dynamodbTypes.AttributeValueMemberS{Value: "2022-03-12T01:01:01Z"},
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
			client: func(t *testing.T) api.GetItemAPI {
				t.Helper()
				return mockutil.MockGetItemAPI(func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
					return &dynamodb.GetItemOutput{
						Item: map[string]dynamodbTypes.AttributeValue{
							"MessageID":     params.Key["MessageID"],
							"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: "inbox#2023-02"},
						},
					}, nil
				})
			},
			messageID:   "exampleMessageID",
			expected:    nil,
			expectedErr: api.ErrNotFound,
		},
		{
			client: func(t *testing.T) api.GetItemAPI {
				t.Helper()
				return mockutil.MockGetItemAPI(func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
					return &dynamodb.GetItemOutput{
						Item: map[string]dynamodbTypes.AttributeValue{
							"MessageID":     params.Key["MessageID"],
							"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: "invalid#2023-02"},
						},
					}, nil
				})
			},
			messageID:   "exampleMessageID",
			expected:    nil,
			expectedErr: format.ErrInvalidEmailType,
		},
		{
			client: func(t *testing.T) api.GetItemAPI {
				t.Helper()
				return mockutil.MockGetItemAPI(func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
					return &dynamodb.GetItemOutput{
						Item: nil,
					}, nil
				})
			},
			messageID:   "exampleMessageID",
			expected:    nil,
			expectedErr: api.ErrNotFound,
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

func TestGetThreadWithEmails(t *testing.T) {
	env.TableName = "table-for-get-thread-with-emails"
	tests := []struct {
		client      func(t *testing.T) api.GetThreadWithEmailsAPI
		messageID   string
		expected    *Thread
		expectedErr error
	}{
		{
			client: func(t *testing.T) api.GetThreadWithEmailsAPI {
				return mockutil.MockGetThreadWithEmailsAPI{
					MockGetItem: func(_ context.Context, params *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
						t.Helper()
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
								"MessageID":     params.Key["MessageID"],
								"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: "thread#2023-02"},
								"Subject":       &dynamodbTypes.AttributeValueMemberS{Value: "subject"},
								"EmailIDs": &dynamodbTypes.AttributeValueMemberL{
									Value: []dynamodbTypes.AttributeValue{
										&dynamodbTypes.AttributeValueMemberS{Value: "id-1"},
										&dynamodbTypes.AttributeValueMemberS{Value: "id-2"},
									},
								},
								"TimeUpdated": &dynamodbTypes.AttributeValueMemberS{Value: "2023-02-18T01:01:01Z"},
							},
						}, nil
					},
					MockBatchGetItem: func(_ context.Context, params *dynamodb.BatchGetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error) {
						t.Helper()
						assert.NotNil(t, params.RequestItems)
						assert.Len(t, params.RequestItems, 1)
						assert.Len(t, params.RequestItems[env.TableName].Keys, 2)
						assert.Equal(t,
							params.RequestItems[env.TableName].Keys[0]["MessageID"].(*dynamodbTypes.AttributeValueMemberS).Value,
							"id-1",
						)
						assert.Equal(t,
							params.RequestItems[env.TableName].Keys[1]["MessageID"].(*dynamodbTypes.AttributeValueMemberS).Value,
							"id-2",
						)

						return &dynamodb.BatchGetItemOutput{
							Responses: map[string][]map[string]dynamodbTypes.AttributeValue{
								env.TableName: {
									{
										"MessageID":     &dynamodbTypes.AttributeValueMemberS{Value: "id-1"},
										"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: "inbox#2023-02"},
										"DateTime":      &dynamodbTypes.AttributeValueMemberS{Value: "18-01:01:01"},
										"Subject":       &dynamodbTypes.AttributeValueMemberS{Value: "subject"},
										"From":          &dynamodbTypes.AttributeValueMemberSS{Value: []string{"example@example.com"}},
										"To":            &dynamodbTypes.AttributeValueMemberSS{Value: []string{"example@example.com"}},
									},
									{
										"MessageID":     &dynamodbTypes.AttributeValueMemberS{Value: "id-2"},
										"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: "inbox#2023-02"},
										"DateTime":      &dynamodbTypes.AttributeValueMemberS{Value: "18-01:01:01"},
										"Subject":       &dynamodbTypes.AttributeValueMemberS{Value: "subject"},
										"From":          &dynamodbTypes.AttributeValueMemberSS{Value: []string{"example@example.com"}},
										"To":            &dynamodbTypes.AttributeValueMemberSS{Value: []string{"example@example.com"}},
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
				Emails: []email.GetResult{
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
			client: func(t *testing.T) api.GetThreadWithEmailsAPI {
				t.Helper()
				return mockutil.MockGetThreadWithEmailsAPI{
					MockGetItem: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
						return &dynamodb.GetItemOutput{}, nil
					},
				}
			},
			messageID:   "exampleMessageID",
			expected:    nil,
			expectedErr: api.ErrNotFound,
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
	id := idutil.GenerateThreadID()
	assert.NotEmpty(t, id)
	assert.Len(t, id, 36-4) // minus the 4 dashes
	assert.NotContains(t, id, "-")
}

func TestStoreEmailWithExistingThread(t *testing.T) {
	env.TableName = "table-for-store-email-with-existing-thread"
	tests := []struct {
		client            func(t *testing.T) api.TransactWriteItemsAPI
		threadID          string
		email             map[string]dynamodbTypes.AttributeValue
		timeReceived      string
		previousMessageID string
		expectedErr       error
	}{
		{
			client: func(t *testing.T) api.TransactWriteItemsAPI {
				return mockutil.MockTransactWriteItemAPI(func(ctx context.Context, params *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
					t.Helper()

					for _, item := range params.TransactItems {
						if item.Put != nil {
							assert.Equal(t, env.TableName, *item.Put.TableName)
							assert.Equal(t, map[string]dynamodbTypes.AttributeValue{
								"MessageID":      &dynamodbTypes.AttributeValueMemberS{Value: "exampleMessageID"},
								"IsThreadLatest": &dynamodbTypes.AttributeValueMemberBOOL{Value: true},
							}, item.Put.Item)
						}
						if item.Update != nil {
							assert.Equal(t, env.TableName, *item.Update.TableName)
							assert.IsType(t, item.Update.Key["MessageID"], &dynamodbTypes.AttributeValueMemberS{})

							if item.Update.Key["MessageID"].(*dynamodbTypes.AttributeValueMemberS).Value == "exampleThreadID" {
								assert.Equal(t, "SET #emails = list_append(#emails, :emails), #timeUpdated = :timeUpdated", *item.Update.UpdateExpression)
								assert.Equal(t, map[string]string{
									"#emails":      "EmailIDs",
									"#timeUpdated": "TimeUpdated",
								}, item.Update.ExpressionAttributeNames)
								assert.Equal(t, map[string]dynamodbTypes.AttributeValue{
									":emails": &dynamodbTypes.AttributeValueMemberL{
										Value: []dynamodbTypes.AttributeValue{
											&dynamodbTypes.AttributeValueMemberS{Value: "exampleMessageID"},
										},
									},
									":timeUpdated": &dynamodbTypes.AttributeValueMemberS{Value: "2023-02-18T01:01:01Z"},
								}, item.Update.ExpressionAttributeValues)
							} else {
								assert.Equal(t, "examplePreviousMessageID", item.Update.Key["MessageID"].(*dynamodbTypes.AttributeValueMemberS).Value)
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
			email: map[string]dynamodbTypes.AttributeValue{
				"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: "exampleMessageID"},
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
	env.TableName = "table-for-store-email-with-existing-thread"
	tests := []struct {
		client          func(t *testing.T) api.TransactWriteItemsAPI
		threadID        string
		email           map[string]dynamodbTypes.AttributeValue
		CreatingEmailID string
		CreatingSubject string
		CreatingTime    string
		TimeReceived    string
		expectedErr     error
	}{
		{
			client: func(t *testing.T) api.TransactWriteItemsAPI {
				return mockutil.MockTransactWriteItemAPI(func(ctx context.Context, params *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
					t.Helper()
					for _, item := range params.TransactItems {
						if item.Put != nil {
							assert.Equal(t, env.TableName, *item.Put.TableName)

							switch {
							case item.Put.Item["MessageID"].(*dynamodbTypes.AttributeValueMemberS).Value == "exampleThreadID":
								assert.Equal(t, map[string]dynamodbTypes.AttributeValue{
									"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: "exampleThreadID"},
									"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{
										Value: "thread#2023-02",
									},
									"EmailIDs": &dynamodbTypes.AttributeValueMemberL{
										Value: []dynamodbTypes.AttributeValue{
											&dynamodbTypes.AttributeValueMemberS{Value: "exampleCreatingEmailID"},
											&dynamodbTypes.AttributeValueMemberS{Value: "exampleMessageID"},
										},
									},
									"TimeUpdated": &dynamodbTypes.AttributeValueMemberS{Value: "2023-02-19T01:01:01Z"},
									"Subject":     &dynamodbTypes.AttributeValueMemberS{Value: "exampleCreatingSubject"},
								}, item.Put.Item)
							case item.Put.Item["MessageID"].(*dynamodbTypes.AttributeValueMemberS).Value == "exampleMessageID":
								assert.Equal(t, map[string]dynamodbTypes.AttributeValue{
									"MessageID":      &dynamodbTypes.AttributeValueMemberS{Value: "exampleMessageID"},
									"IsThreadLatest": &dynamodbTypes.AttributeValueMemberBOOL{Value: true},
								}, item.Put.Item)
							default:
								assert.Fail(t, "unexpected item", item.Put.Item)
							}
						}
						if item.Update != nil {
							assert.Equal(t, env.TableName, *item.Update.TableName)
							assert.IsType(t, item.Update.Key["MessageID"], &dynamodbTypes.AttributeValueMemberS{})
							assert.Equal(t,
								item.Update.Key["MessageID"].(*dynamodbTypes.AttributeValueMemberS).Value,
								"exampleCreatingEmailID",
							)
							assert.Equal(t, "SET #threadID = :threadID", *item.Update.UpdateExpression)
							assert.Equal(t, map[string]string{
								"#threadID": "ThreadID",
							}, item.Update.ExpressionAttributeNames)
							assert.Equal(t, map[string]dynamodbTypes.AttributeValue{
								":threadID": &dynamodbTypes.AttributeValueMemberS{Value: "exampleThreadID"},
							}, item.Update.ExpressionAttributeValues)
						}
					}

					return &dynamodb.TransactWriteItemsOutput{
						ResultMetadata: middleware.Metadata{},
					}, nil
				})
			},
			threadID: "exampleThreadID",
			email: map[string]dynamodbTypes.AttributeValue{
				"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: "exampleMessageID"},
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
