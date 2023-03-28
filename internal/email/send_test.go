package email

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/stretchr/testify/assert"
)

type mockSendEmailAPI struct {
	mockGetItem           mockGetItemAPI
	mockTransactWriteItem mockTransactWriteItemAPI
	mockSendEmail         func(ctx context.Context, params *sesv2.SendEmailInput, optFns ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error)
}

func (m mockSendEmailAPI) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return m.mockGetItem(ctx, params, optFns...)
}

func (m mockSendEmailAPI) TransactWriteItems(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	return m.mockTransactWriteItem(ctx, params, optFns...)
}

func (m mockSendEmailAPI) SendEmail(ctx context.Context, params *sesv2.SendEmailInput, optFns ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
	return m.mockSendEmail(ctx, params, optFns...)
}

func TestSend(t *testing.T) {
	tests := []struct {
		client      func(t *testing.T) GetAndSendEmailAPI
		messageID   string
		expectedErr error
	}{
		{
			client: func(t *testing.T) GetAndSendEmailAPI {
				return mockSendEmailAPI{
					mockGetItem: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
						return &dynamodb.GetItemOutput{
							Item: map[string]dynamodbtypes.AttributeValue{
								"MessageID":     &dynamodbtypes.AttributeValueMemberS{Value: "draft-id"},
								"TypeYearMonth": &dynamodbtypes.AttributeValueMemberS{Value: "draft#2022-03"},
								"DateTime":      &dynamodbtypes.AttributeValueMemberS{Value: "12-01:01:01"},
								"Subject":       &dynamodbtypes.AttributeValueMemberS{Value: "subject"},
								"DateSent":      &dynamodbtypes.AttributeValueMemberS{Value: "2022-03-12T01:01:01Z"},
								"Source":        &dynamodbtypes.AttributeValueMemberS{Value: "example@example.com"},
								"Destination":   &dynamodbtypes.AttributeValueMemberSS{Value: []string{"example@example.com"}},
								"From":          &dynamodbtypes.AttributeValueMemberSS{Value: []string{"example@example.com"}},
								"To":            &dynamodbtypes.AttributeValueMemberSS{Value: []string{"example@example.com"}},
								"ReturnPath":    &dynamodbtypes.AttributeValueMemberS{Value: "example@example.com"},
								"Text":          &dynamodbtypes.AttributeValueMemberS{Value: "text"},
								"HTML":          &dynamodbtypes.AttributeValueMemberS{Value: "html"},
							},
						}, nil
					},
					mockSendEmail: func(ctx context.Context, params *sesv2.SendEmailInput, optFns ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
						return &sesv2.SendEmailOutput{
							MessageId: aws.String("newID"),
						}, nil
					},
					mockTransactWriteItem: func(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
						return &dynamodb.TransactWriteItemsOutput{}, nil
					},
				}
			},
			messageID: "draft-id",
		},
		{
			client: func(t *testing.T) GetAndSendEmailAPI {
				return mockSendEmailAPI{}
			},
			messageID:   "invalid-id",
			expectedErr: ErrEmailIsNotDraft,
		},
		{
			client: func(t *testing.T) GetAndSendEmailAPI {
				return mockSendEmailAPI{
					mockGetItem: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
						return &dynamodb.GetItemOutput{
							Item: map[string]dynamodbtypes.AttributeValue{},
						}, ErrNotFound
					},
				}
			},
			messageID:   "draft-id",
			expectedErr: ErrNotFound,
		},
		{
			client: func(t *testing.T) GetAndSendEmailAPI {
				return mockSendEmailAPI{
					mockGetItem: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
						return &dynamodb.GetItemOutput{
							Item: map[string]dynamodbtypes.AttributeValue{
								"MessageID":     &dynamodbtypes.AttributeValueMemberS{Value: "draft-id"},
								"TypeYearMonth": &dynamodbtypes.AttributeValueMemberS{Value: "draft#2022-03"},
								"DateTime":      &dynamodbtypes.AttributeValueMemberS{Value: "12-01:01:01"},
								"Subject":       &dynamodbtypes.AttributeValueMemberS{Value: "subject"},
								"DateSent":      &dynamodbtypes.AttributeValueMemberS{Value: "2022-03-12T01:01:01Z"},
								"Source":        &dynamodbtypes.AttributeValueMemberS{Value: "example@example.com"},
								"Destination":   &dynamodbtypes.AttributeValueMemberSS{Value: []string{"example@example.com"}},
								"From":          &dynamodbtypes.AttributeValueMemberSS{Value: []string{"example@example.com"}},
								"To":            &dynamodbtypes.AttributeValueMemberSS{Value: []string{"example@example.com"}},
								"ReturnPath":    &dynamodbtypes.AttributeValueMemberS{Value: "example@example.com"},
								"Text":          &dynamodbtypes.AttributeValueMemberS{Value: "text"},
								"HTML":          &dynamodbtypes.AttributeValueMemberS{Value: "html"},
							},
						}, nil
					},
					mockSendEmail: func(ctx context.Context, params *sesv2.SendEmailInput, optFns ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
						return &sesv2.SendEmailOutput{}, errors.New("1")
					},
					mockTransactWriteItem: func(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
						return &dynamodb.TransactWriteItemsOutput{}, nil
					},
				}
			},
			messageID:   "draft-id",
			expectedErr: errors.New("1"),
		},
		{
			client: func(t *testing.T) GetAndSendEmailAPI {
				return mockSendEmailAPI{
					mockGetItem: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
						return &dynamodb.GetItemOutput{
							Item: map[string]dynamodbtypes.AttributeValue{
								"MessageID":     &dynamodbtypes.AttributeValueMemberS{Value: "draft-id"},
								"TypeYearMonth": &dynamodbtypes.AttributeValueMemberS{Value: "draft#2022-03"},
								"DateTime":      &dynamodbtypes.AttributeValueMemberS{Value: "12-01:01:01"},
								"Subject":       &dynamodbtypes.AttributeValueMemberS{Value: "subject"},
								"DateSent":      &dynamodbtypes.AttributeValueMemberS{Value: "2022-03-12T01:01:01Z"},
								"Source":        &dynamodbtypes.AttributeValueMemberS{Value: "example@example.com"},
								"Destination":   &dynamodbtypes.AttributeValueMemberSS{Value: []string{"example@example.com"}},
								"From":          &dynamodbtypes.AttributeValueMemberSS{Value: []string{"example@example.com"}},
								"To":            &dynamodbtypes.AttributeValueMemberSS{Value: []string{"example@example.com"}},
								"ReturnPath":    &dynamodbtypes.AttributeValueMemberS{Value: "example@example.com"},
								"Text":          &dynamodbtypes.AttributeValueMemberS{Value: "text"},
								"HTML":          &dynamodbtypes.AttributeValueMemberS{Value: "html"},
							},
						}, nil
					},
					mockSendEmail: func(ctx context.Context, params *sesv2.SendEmailInput, optFns ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
						return &sesv2.SendEmailOutput{
							MessageId: aws.String("newID"),
						}, nil
					},
					mockTransactWriteItem: func(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
						return &dynamodb.TransactWriteItemsOutput{}, errors.New("2")
					},
				}
			},
			messageID:   "draft-id",
			expectedErr: errors.New("2"),
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()
			result, err := Send(ctx, test.client(t), test.messageID)
			assert.Equal(t, test.expectedErr, err)
			if test.expectedErr == nil {
				assert.NotNil(t, result)
				assert.NotEmpty(t, result.MessageID)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestSendEmailViaSES(t *testing.T) {
	tests := []struct {
		client            func(t *testing.T, email *EmailInput) SendEmailAPI
		email             *EmailInput
		expectedMessageID string
		expectedErr       error
	}{
		{
			client: func(t *testing.T, email *EmailInput) SendEmailAPI {
				return mockSendEmailAPI{
					mockSendEmail: func(ctx context.Context, params *sesv2.SendEmailInput, optFns ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
						t.Helper()

						assert.Nil(t, params.Content.Raw)
						assert.Nil(t, params.Content.Template)
						assert.Equal(t, email.HTML, *params.Content.Simple.Body.Html.Data)
						assert.Equal(t, "UTF-8", *params.Content.Simple.Body.Html.Charset)
						assert.Equal(t, email.Text, *params.Content.Simple.Body.Text.Data)
						assert.Equal(t, "UTF-8", *params.Content.Simple.Body.Text.Charset)

						assert.Equal(t, email.Subject, *params.Content.Simple.Subject.Data)
						assert.Equal(t, "UTF-8", *params.Content.Simple.Subject.Charset)

						assert.Equal(t, email.To, params.Destination.ToAddresses)
						assert.Equal(t, email.Cc, params.Destination.CcAddresses)
						assert.Equal(t, email.Bcc, params.Destination.BccAddresses)

						assert.Equal(t, email.From[0], *params.FromEmailAddress)
						assert.Equal(t, email.ReplyTo, params.ReplyToAddresses)

						return &sesv2.SendEmailOutput{
							MessageId: aws.String("newMessageID"),
						}, nil
					},
				}
			},
			email: &EmailInput{
				MessageID: "exampleMessageID",
				Subject:   "subject",
				To:        []string{"example@example.com"},
				Cc:        []string{"example@example.com"},
				Bcc:       []string{"example@example.com"},
				From:      []string{"example@example.com"},
				ReplyTo:   []string{"example@example.com"},
				HTML:      "html",
				Text:      "text",
			},
			expectedMessageID: "newMessageID",
		},
		{
			client: func(t *testing.T, email *EmailInput) SendEmailAPI {
				return mockSendEmailAPI{
					mockSendEmail: func(ctx context.Context, params *sesv2.SendEmailInput, optFns ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
						return &sesv2.SendEmailOutput{}, ErrEmailIsNotDraft
					},
				}
			},
			email: &EmailInput{
				From: []string{""},
			},
			expectedErr: ErrEmailIsNotDraft,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()
			messageID, err := sendEmailViaSES(ctx, test.client(t, test.email), test.email)
			assert.Equal(t, test.expectedMessageID, messageID)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}

func TestMarkEmailAsSent(t *testing.T) {
	tests := []struct {
		client       func(t *testing.T) SendEmailAPI
		oldMessageID string
		email        *EmailInput
		expectedErr  error
	}{
		{
			client: func(t *testing.T) SendEmailAPI {
				return mockSendEmailAPI{
					mockTransactWriteItem: func(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
						t.Helper()

						assert.Len(t, params.TransactItems, 2)
						for _, item := range params.TransactItems {
							if item.Delete != nil {
								assert.Len(t, item.Delete.Key, 1)
								assert.Equal(t, "oldID", item.Delete.Key["MessageID"].(*dynamodbTypes.AttributeValueMemberS).Value)
							}
							if item.Put != nil {
								assert.NotNil(t, item.Put.Item)
								assert.Equal(t, "newID", item.Put.Item["MessageID"].(*dynamodbTypes.AttributeValueMemberS).Value)
								assert.Contains(t, item.Put.Item["TypeYearMonth"].(*dynamodbTypes.AttributeValueMemberS).Value, "sent#")
							}
						}

						return &dynamodb.TransactWriteItemsOutput{}, nil
					},
				}
			},
			oldMessageID: "oldID",
			email: &EmailInput{
				MessageID: "newID",
				Subject:   "subject",
				To:        []string{"example@example.com"},
				Cc:        []string{"example@example.com"},
				Bcc:       []string{"example@example.com"},
				From:      []string{"example@example.com"},
				ReplyTo:   []string{"example@example.com"},
				HTML:      "html",
				Text:      "text",
			},
		},
		{
			client: func(t *testing.T) SendEmailAPI {
				return mockSendEmailAPI{
					mockTransactWriteItem: func(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
						return &dynamodb.TransactWriteItemsOutput{}, nil
					},
				}
			},
			email: &EmailInput{
				MessageID: "newID",
				Subject:   "subject",
				To:        []string{"example@example.com"},
				Cc:        []string{"example@example.com"},
				Bcc:       []string{"example@example.com"},
				From:      []string{"example@example.com"},
				ReplyTo:   []string{"example@example.com"},
				HTML:      "html",
				Text:      "text",
			},
			expectedErr: ErrNotFound,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()
			err := markEmailAsSent(ctx, test.client(t), test.oldMessageID, test.email)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
