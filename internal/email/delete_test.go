package email

import (
	"context"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/model"
	"github.com/harryzcy/mailbox/internal/platform"
	"github.com/stretchr/testify/assert"
)

type mockDeleteItemAPI struct {
	mockGetItem      func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	mockDeleteItem   func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	mockDeleteObject func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	mockUpdateItem   func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
}

// GetItem returns a trashed email outside a thread unless mockGetItem is set
func (m mockDeleteItemAPI) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	if m.mockGetItem == nil {
		return &dynamodb.GetItemOutput{Item: map[string]dynamodbTypes.AttributeValue{
			"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: "inbox#2023-02"},
			"TrashedTime":   &dynamodbTypes.AttributeValueMemberS{Value: "2023-02-01T01:01:01Z"},
		}}, nil
	}
	return m.mockGetItem(ctx, params, optFns...)
}

func (m mockDeleteItemAPI) DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	return m.mockDeleteItem(ctx, params, optFns...)
}

func (m mockDeleteItemAPI) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	return m.mockUpdateItem(ctx, params, optFns...)
}

func (m mockDeleteItemAPI) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return m.mockDeleteObject(ctx, params, optFns...)
}

func TestDelete(t *testing.T) {
	env.TableName = "table-for-delete"
	tests := []struct {
		client      func(t *testing.T) platform.DeleteEmailAPI
		messageID   string
		expectedErr error
	}{
		{
			client: func(t *testing.T) platform.DeleteEmailAPI {
				return mockDeleteItemAPI{
					mockDeleteItem: func(_ context.Context, params *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
						t.Helper()

						assert.Equal(t, env.TableName, *params.TableName)
						assert.Len(t, params.Key, 1)
						assert.IsType(t, params.Key["MessageID"], &dynamodbTypes.AttributeValueMemberS{})
						assert.Equal(t,
							params.Key["MessageID"].(*dynamodbTypes.AttributeValueMemberS).Value,
							"exampleMessageID",
						)

						assert.Equal(t, "begins_with(TypeYearMonth, :v_type) OR (attribute_exists(TrashedTime) AND (attribute_not_exists(ThreadID) OR size(ThreadID) = :zero))",
							*params.ConditionExpression)
						assert.Len(t, params.ExpressionAttributeValues, 2)
						assert.Contains(t, params.ExpressionAttributeValues, ":v_type")
						assert.Equal(t, params.ExpressionAttributeValues[":v_type"].(*dynamodbTypes.AttributeValueMemberS).Value,
							model.EmailTypeDraft)
						assert.Equal(t, "0", params.ExpressionAttributeValues[":zero"].(*dynamodbTypes.AttributeValueMemberN).Value)

						return &dynamodb.DeleteItemOutput{}, nil
					},
					mockDeleteObject: func(_ context.Context, _ *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
						return &s3.DeleteObjectOutput{}, nil
					},
				}
			},
			messageID: "exampleMessageID",
		},
		{
			client: func(t *testing.T) platform.DeleteEmailAPI {
				t.Helper()
				return mockDeleteItemAPI{
					mockDeleteItem: func(_ context.Context, _ *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
						return &dynamodb.DeleteItemOutput{}, &dynamodbTypes.ConditionalCheckFailedException{}
					},
					mockDeleteObject: func(_ context.Context, _ *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
						return &s3.DeleteObjectOutput{}, nil
					},
				}
			},
			expectedErr: &platform.NotTrashedError{Type: "email"},
		},
		{
			client: func(t *testing.T) platform.DeleteEmailAPI {
				t.Helper()
				return mockDeleteItemAPI{
					mockDeleteItem: func(_ context.Context, _ *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
						return &dynamodb.DeleteItemOutput{}, &platform.NotTrashedError{Type: "email"}
					},
					mockDeleteObject: func(_ context.Context, _ *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
						return &s3.DeleteObjectOutput{}, nil
					},
				}
			},
			expectedErr: &platform.NotTrashedError{Type: "email"},
		},
		{
			// S3 delete fails: the email is kept so the delete can be retried
			client: func(t *testing.T) platform.DeleteEmailAPI {
				t.Helper()
				return mockDeleteItemAPI{
					mockDeleteItem: func(_ context.Context, _ *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
						t.Error("email deleted from DynamoDB after the S3 delete failed")
						return &dynamodb.DeleteItemOutput{}, nil
					},
					mockDeleteObject: func(_ context.Context, _ *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
						return &s3.DeleteObjectOutput{}, platform.ErrNotFound
					},
				}
			},
			expectedErr: platform.ErrNotFound,
		},
		{
			// reply draft, removed from its thread
			client: func(t *testing.T) platform.DeleteEmailAPI {
				t.Helper()
				return mockDeleteItemAPI{
					mockDeleteItem: func(_ context.Context, _ *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
						return &dynamodb.DeleteItemOutput{
							Attributes: map[string]dynamodbTypes.AttributeValue{
								"ThreadID": &dynamodbTypes.AttributeValueMemberS{Value: "thread-id"},
							},
						}, nil
					},
					mockUpdateItem: func(_ context.Context, params *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
						assert.Equal(t, "thread-id", params.Key["MessageID"].(*dynamodbTypes.AttributeValueMemberS).Value)
						assert.Equal(t, "REMOVE DraftID", *params.UpdateExpression)
						assert.Equal(t, "DraftID = :draftID", *params.ConditionExpression)
						assert.Equal(t, "draft-id", params.ExpressionAttributeValues[":draftID"].(*dynamodbTypes.AttributeValueMemberS).Value)
						return &dynamodb.UpdateItemOutput{}, nil
					},
					mockDeleteObject: func(_ context.Context, _ *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
						return &s3.DeleteObjectOutput{}, nil
					},
				}
			},
			messageID: "draft-id",
		},
		{
			// reply draft, thread already points to a newer draft
			client: func(t *testing.T) platform.DeleteEmailAPI {
				t.Helper()
				return mockDeleteItemAPI{
					mockDeleteItem: func(_ context.Context, _ *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
						return &dynamodb.DeleteItemOutput{
							Attributes: map[string]dynamodbTypes.AttributeValue{
								"ThreadID": &dynamodbTypes.AttributeValueMemberS{Value: "thread-id"},
							},
						}, nil
					},
					mockUpdateItem: func(_ context.Context, _ *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
						return nil, &dynamodbTypes.ConditionalCheckFailedException{}
					},
					mockDeleteObject: func(_ context.Context, _ *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
						return &s3.DeleteObjectOutput{}, nil
					},
				}
			},
			messageID: "draft-id",
		},
		{
			// trashed email stored with an empty ThreadID: no thread to update
			client: func(t *testing.T) platform.DeleteEmailAPI {
				t.Helper()
				deletedObject := false
				t.Cleanup(func() { assert.True(t, deletedObject) })
				return mockDeleteItemAPI{
					mockDeleteItem: func(_ context.Context, _ *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
						return &dynamodb.DeleteItemOutput{
							Attributes: map[string]dynamodbTypes.AttributeValue{
								"ThreadID": &dynamodbTypes.AttributeValueMemberS{Value: ""},
							},
						}, nil
					},
					mockDeleteObject: func(_ context.Context, _ *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
						deletedObject = true
						return &s3.DeleteObjectOutput{}, nil
					},
				}
			},
			messageID: "exampleMessageID",
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()

			err := Delete(ctx, test.client(t), test.messageID)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}

func TestDelete_NotDeletable(t *testing.T) {
	env.TableName = "table-for-delete-check"
	trashed := &dynamodbTypes.AttributeValueMemberS{Value: "2023-02-01T01:01:01Z"}
	inbox := &dynamodbTypes.AttributeValueMemberS{Value: "inbox#2023-02"}

	tests := []struct {
		name      string
		item      map[string]dynamodbTypes.AttributeValue
		deletable bool
	}{
		{name: "trashed", item: map[string]dynamodbTypes.AttributeValue{"TypeYearMonth": inbox, "TrashedTime": trashed}, deletable: true},
		{
			name: "trashed, stored with an empty ThreadID",
			item: map[string]dynamodbTypes.AttributeValue{
				"TypeYearMonth": inbox, "TrashedTime": trashed, "ThreadID": &dynamodbTypes.AttributeValueMemberS{Value: ""},
			},
			deletable: true,
		},
		{
			name: "draft in a thread",
			item: map[string]dynamodbTypes.AttributeValue{
				"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: "draft#2023-02"},
				"ThreadID":      &dynamodbTypes.AttributeValueMemberS{Value: "thread-id"},
			},
			deletable: true,
		},
		{name: "not trashed", item: map[string]dynamodbTypes.AttributeValue{"TypeYearMonth": inbox}},
		{
			name: "trashed, in a thread",
			item: map[string]dynamodbTypes.AttributeValue{
				"TypeYearMonth": inbox, "TrashedTime": trashed, "ThreadID": &dynamodbTypes.AttributeValueMemberS{Value: "thread-id"},
			},
		},
		{name: "not found"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deletedObject, deletedItem := false, false
			client := mockDeleteItemAPI{
				mockGetItem: func(_ context.Context, params *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
					assert.True(t, aws.ToBool(params.ConsistentRead))
					return &dynamodb.GetItemOutput{Item: test.item}, nil
				},
				mockDeleteItem: func(_ context.Context, _ *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
					deletedItem = true
					return &dynamodb.DeleteItemOutput{}, nil
				},
				mockUpdateItem: func(_ context.Context, _ *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
					return &dynamodb.UpdateItemOutput{}, nil
				},
				mockDeleteObject: func(_ context.Context, _ *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
					deletedObject = true
					return &s3.DeleteObjectOutput{}, nil
				},
			}

			err := Delete(context.TODO(), client, "exampleMessageID")
			if test.deletable {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, &platform.NotTrashedError{Type: "email"}, err)
			}
			assert.Equal(t, test.deletable, deletedObject)
			assert.Equal(t, test.deletable, deletedItem)
		})
	}
}
