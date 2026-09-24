package thread

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/platform"
	"github.com/stretchr/testify/assert"
)

type mockDeleteThreadAPI struct {
	getItem            func(ctx context.Context, params *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
	transactWriteItems func(ctx context.Context, params *dynamodb.TransactWriteItemsInput) (*dynamodb.TransactWriteItemsOutput, error)
	deleteObject       func(ctx context.Context, params *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error)
}

func (m mockDeleteThreadAPI) GetItem(ctx context.Context, params *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return m.getItem(ctx, params)
}

func (m mockDeleteThreadAPI) TransactWriteItems(ctx context.Context, params *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	return m.transactWriteItems(ctx, params)
}

func (m mockDeleteThreadAPI) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return m.deleteObject(ctx, params)
}

func threadItem(trashed bool, draftID string) map[string]dynamodbTypes.AttributeValue {
	item := map[string]dynamodbTypes.AttributeValue{
		"MessageID":     &dynamodbTypes.AttributeValueMemberS{Value: "exampleThreadID"},
		"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: "thread#2023-02"},
		"EmailIDs": &dynamodbTypes.AttributeValueMemberL{
			Value: []dynamodbTypes.AttributeValue{
				&dynamodbTypes.AttributeValueMemberS{Value: "id-1"},
				&dynamodbTypes.AttributeValueMemberS{Value: "id-2"},
			},
		},
	}
	if trashed {
		item["TrashedTime"] = &dynamodbTypes.AttributeValueMemberS{Value: "2023-02-01T01:01:01Z"}
	}
	if draftID != "" {
		item["DraftID"] = &dynamodbTypes.AttributeValueMemberS{Value: draftID}
	}
	return item
}

func TestDelete(t *testing.T) {
	env.TableName = "table-for-delete-thread"
	errUnexpected := errors.New("unexpected call")

	tests := []struct {
		client            func(t *testing.T, deleted *[]string) platform.DeleteThreadAPI
		expectedS3Deletes []string
		expectedErr       error
	}{
		{
			// trashed thread with a draft: deletes the thread, its emails and draft
			client: func(t *testing.T, deleted *[]string) platform.DeleteThreadAPI {
				return mockDeleteThreadAPI{
					getItem: func(_ context.Context, _ *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
						return &dynamodb.GetItemOutput{Item: threadItem(true, "draft-1")}, nil
					},
					transactWriteItems: func(_ context.Context, params *dynamodb.TransactWriteItemsInput) (*dynamodb.TransactWriteItemsOutput, error) {
						t.Helper()
						assert.Len(t, params.TransactItems, 4)

						thread := params.TransactItems[0].Delete
						assert.Equal(t, "exampleThreadID", thread.Key["MessageID"].(*dynamodbTypes.AttributeValueMemberS).Value)
						assert.Equal(t, "attribute_exists(TrashedTime)", *thread.ConditionExpression)

						for i, id := range []string{"id-1", "id-2", "draft-1"} {
							item := params.TransactItems[i+1].Delete
							assert.Equal(t, env.TableName, *item.TableName)
							assert.Equal(t, id, item.Key["MessageID"].(*dynamodbTypes.AttributeValueMemberS).Value)
							assert.Equal(t, "ThreadID = :threadID", *item.ConditionExpression)
							assert.Equal(t, "exampleThreadID", item.ExpressionAttributeValues[":threadID"].(*dynamodbTypes.AttributeValueMemberS).Value)
						}
						return &dynamodb.TransactWriteItemsOutput{}, nil
					},
					deleteObject: func(_ context.Context, params *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
						*deleted = append(*deleted, *params.Key)
						return &s3.DeleteObjectOutput{}, nil
					},
				}
			},
			expectedS3Deletes: []string{"id-1", "id-2", "draft-1"},
		},
		{
			// thread not trashed: nothing is deleted
			client: func(_ *testing.T, _ *[]string) platform.DeleteThreadAPI {
				return mockDeleteThreadAPI{
					getItem: func(_ context.Context, _ *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
						return &dynamodb.GetItemOutput{Item: threadItem(false, "")}, nil
					},
					transactWriteItems: func(_ context.Context, _ *dynamodb.TransactWriteItemsInput) (*dynamodb.TransactWriteItemsOutput, error) {
						return nil, errUnexpected
					},
					deleteObject: func(_ context.Context, _ *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
						return nil, errUnexpected
					},
				}
			},
			expectedErr: &platform.NotTrashedError{Type: "thread"},
		},
		{
			// thread untrashed between the read and the delete
			client: func(_ *testing.T, _ *[]string) platform.DeleteThreadAPI {
				return mockDeleteThreadAPI{
					getItem: func(_ context.Context, _ *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
						return &dynamodb.GetItemOutput{Item: threadItem(true, "")}, nil
					},
					transactWriteItems: func(_ context.Context, _ *dynamodb.TransactWriteItemsInput) (*dynamodb.TransactWriteItemsOutput, error) {
						return nil, &dynamodbTypes.TransactionCanceledException{
							CancellationReasons: []dynamodbTypes.CancellationReason{
								{Code: aws.String("ConditionalCheckFailed")},
								{Code: aws.String("None")},
								{Code: aws.String("None")},
							},
						}
					},
					deleteObject: func(_ context.Context, _ *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
						return nil, errUnexpected
					},
				}
			},
			expectedErr: &platform.NotTrashedError{Type: "thread"},
		},
		{
			// thread not found
			client: func(_ *testing.T, _ *[]string) platform.DeleteThreadAPI {
				return mockDeleteThreadAPI{
					getItem: func(_ context.Context, _ *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
						return &dynamodb.GetItemOutput{}, nil
					},
				}
			},
			expectedErr: platform.ErrNotFound,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var deleted []string
			err := Delete(context.TODO(), test.client(t, &deleted), "exampleThreadID")
			assert.Equal(t, test.expectedErr, err)
			assert.Equal(t, test.expectedS3Deletes, deleted)
		})
	}
}
