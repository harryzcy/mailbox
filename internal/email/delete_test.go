package email

import (
	"context"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/harryzcy/mailbox/internal/api"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/stretchr/testify/assert"
)

type mockDeleteItemAPI struct {
	mockDeleteItem   func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	mockDeleteObject func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

func (m mockDeleteItemAPI) DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	return m.mockDeleteItem(ctx, params, optFns...)
}

func (m mockDeleteItemAPI) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return m.mockDeleteObject(ctx, params, optFns...)
}

func TestDelete(t *testing.T) {
	env.TableName = "table-for-delete"
	tests := []struct {
		client      func(t *testing.T) api.DeleteItemAPI
		messageID   string
		expectedErr error
	}{
		{
			client: func(t *testing.T) api.DeleteItemAPI {
				return mockDeleteItemAPI{
					mockDeleteItem: func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
						t.Helper()

						assert.Equal(t, env.TableName, *params.TableName)
						assert.Len(t, params.Key, 1)
						assert.IsType(t, params.Key["MessageID"], &types.AttributeValueMemberS{})
						assert.Equal(t,
							params.Key["MessageID"].(*types.AttributeValueMemberS).Value,
							"exampleMessageID",
						)

						assert.Equal(t, "(attribute_exists(TrashedTime) OR begins_with(TypeYearMonth, :v_type)) AND attribute_not_exists(ThreadID)",
							*params.ConditionExpression)
						assert.Len(t, params.ExpressionAttributeValues, 1)
						assert.Contains(t, params.ExpressionAttributeValues, ":v_type")
						assert.Equal(t, params.ExpressionAttributeValues[":v_type"].(*types.AttributeValueMemberS).Value,
							EmailTypeDraft)

						return &dynamodb.DeleteItemOutput{}, nil
					},
					mockDeleteObject: func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
						return &s3.DeleteObjectOutput{}, nil
					},
				}
			},
			messageID: "exampleMessageID",
		},
		{
			client: func(t *testing.T) api.DeleteItemAPI {
				return mockDeleteItemAPI{
					mockDeleteItem: func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
						return &dynamodb.DeleteItemOutput{}, &types.ConditionalCheckFailedException{}
					},
					mockDeleteObject: func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
						return &s3.DeleteObjectOutput{}, nil
					},
				}
			},
			expectedErr: ErrNotTrashed,
		},
		{
			client: func(t *testing.T) api.DeleteItemAPI {
				return mockDeleteItemAPI{
					mockDeleteItem: func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
						return &dynamodb.DeleteItemOutput{}, ErrNotTrashed
					},
					mockDeleteObject: func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
						return &s3.DeleteObjectOutput{}, nil
					},
				}
			},
			expectedErr: ErrNotTrashed,
		},
		{
			client: func(t *testing.T) api.DeleteItemAPI {
				return mockDeleteItemAPI{
					mockDeleteItem: func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
						return &dynamodb.DeleteItemOutput{}, nil
					},
					mockDeleteObject: func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
						return &s3.DeleteObjectOutput{}, ErrNotFound
					},
				}
			},
			expectedErr: ErrNotFound,
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
