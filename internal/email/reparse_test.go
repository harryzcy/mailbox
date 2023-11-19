package email

import (
	"context"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/harryzcy/mailbox/internal/api"
	"github.com/stretchr/testify/assert"
)

type mockReparseEmailAPI struct {
	mockGetObject  func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	mockUpdateItem func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
}

func (m mockReparseEmailAPI) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return m.mockGetObject(ctx, params, optFns...)
}

func (m mockReparseEmailAPI) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	return m.mockUpdateItem(ctx, params, optFns...)
}

func TestReparse(t *testing.T) {
	exampleMessageID := "test"
	raw := `From: user@inbucket.org
Subject: Example message
Content-Type: multipart/alternative; boundary=Enmime-100

--Enmime-100
Content-Type: text/plain
X-Comment: part1

hello!
--Enmime-100--`
	tests := []struct {
		client      func(t *testing.T) api.ReparseEmailAPI
		messageID   string
		expectedErr error
	}{
		{
			client: func(t *testing.T) api.ReparseEmailAPI {
				return mockReparseEmailAPI{
					mockGetObject: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
						t.Helper()
						assert.Equal(t, exampleMessageID, *params.Key)
						return &s3.GetObjectOutput{
							Body: io.NopCloser(strings.NewReader(raw)),
						}, nil
					},
					mockUpdateItem: func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
						text := "hello!"
						html := ""
						assert.EqualValues(t, &types.AttributeValueMemberS{Value: exampleMessageID}, (*params).Key["MessageID"])
						assert.Equal(t, &types.AttributeValueMemberS{Value: text}, (*params).ExpressionAttributeValues[":text"])
						assert.Equal(t, &types.AttributeValueMemberS{Value: html}, (*params).ExpressionAttributeValues[":html"])
						assert.Empty(t, (*params).ExpressionAttributeValues[":attachments"].(*types.AttributeValueMemberL).Value)
						assert.Empty(t, (*params).ExpressionAttributeValues[":inlines"].(*types.AttributeValueMemberL).Value)
						assert.Empty(t, (*params).ExpressionAttributeValues[":others"].(*types.AttributeValueMemberL).Value)

						return &dynamodb.UpdateItemOutput{}, nil
					},
				}
			},
			messageID: exampleMessageID,
		},
		{
			client: func(t *testing.T) api.ReparseEmailAPI {
				return mockReparseEmailAPI{
					mockGetObject: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
						t.Helper()
						return &s3.GetObjectOutput{}, api.ErrInvalidInput
					},
				}
			},
			messageID:   exampleMessageID,
			expectedErr: api.ErrInvalidInput,
		},
		{
			client: func(t *testing.T) api.ReparseEmailAPI {
				return mockReparseEmailAPI{
					mockGetObject: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
						t.Helper()
						return &s3.GetObjectOutput{
							Body: io.NopCloser(strings.NewReader(raw)),
						}, nil
					},
					mockUpdateItem: func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
						return &dynamodb.UpdateItemOutput{}, api.ErrInvalidInput
					},
				}
			},
			messageID:   exampleMessageID,
			expectedErr: api.ErrInvalidInput,
		},
		{
			client: func(t *testing.T) api.ReparseEmailAPI {
				return mockReparseEmailAPI{
					mockGetObject: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
						t.Helper()
						return &s3.GetObjectOutput{
							Body: io.NopCloser(strings.NewReader(raw)),
						}, nil
					},
					mockUpdateItem: func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
						return &dynamodb.UpdateItemOutput{}, &types.ProvisionedThroughputExceededException{}
					},
				}
			},
			messageID:   exampleMessageID,
			expectedErr: api.ErrTooManyRequests,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()
			err := Reparse(ctx, test.client(t), test.messageID)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
