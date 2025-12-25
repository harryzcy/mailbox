package email

import (
	"context"
	"errors"
	"net/mail"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/harryzcy/mailbox/internal/platform"
	"github.com/harryzcy/mailbox/internal/util/mockutil"
	"github.com/stretchr/testify/assert"
)

type mockSendEmailAPI struct {
	mockGetItem           mockGetItemAPI
	mockTransactWriteItem mockutil.MockTransactWriteItemAPI
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
		client      func(t *testing.T) platform.GetAndSendEmailAPI
		messageID   string
		expectedErr error
	}{
		{
			client: func(t *testing.T) platform.GetAndSendEmailAPI {
				t.Helper()
				return mockSendEmailAPI{
					mockGetItem: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
						return &dynamodb.GetItemOutput{
							Item: map[string]dynamodbTypes.AttributeValue{
								"MessageID":     &dynamodbTypes.AttributeValueMemberS{Value: "draft-id"},
								"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: "draft#2022-03"},
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
					},
					mockSendEmail: func(_ context.Context, _ *sesv2.SendEmailInput, _ ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
						return &sesv2.SendEmailOutput{
							MessageId: aws.String("newID"),
						}, nil
					},
					mockTransactWriteItem: func(_ context.Context, _ *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
						return &dynamodb.TransactWriteItemsOutput{}, nil
					},
				}
			},
			messageID: "draft-id",
		},
		{
			client: func(t *testing.T) platform.GetAndSendEmailAPI {
				t.Helper()
				return mockSendEmailAPI{}
			},
			messageID:   "invalid-id",
			expectedErr: platform.ErrEmailIsNotDraft,
		},
		{
			client: func(t *testing.T) platform.GetAndSendEmailAPI {
				t.Helper()
				return mockSendEmailAPI{
					mockGetItem: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
						return &dynamodb.GetItemOutput{
							Item: map[string]dynamodbTypes.AttributeValue{},
						}, platform.ErrNotFound
					},
				}
			},
			messageID:   "draft-id",
			expectedErr: platform.ErrNotFound,
		},
		{
			client: func(t *testing.T) platform.GetAndSendEmailAPI {
				t.Helper()
				return mockSendEmailAPI{
					mockGetItem: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
						return &dynamodb.GetItemOutput{
							Item: map[string]dynamodbTypes.AttributeValue{
								"MessageID":     &dynamodbTypes.AttributeValueMemberS{Value: "draft-id"},
								"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: "draft#2022-03"},
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
					},
					mockSendEmail: func(_ context.Context, _ *sesv2.SendEmailInput, _ ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
						return &sesv2.SendEmailOutput{}, errors.New("1")
					},
					mockTransactWriteItem: func(_ context.Context, _ *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
						return &dynamodb.TransactWriteItemsOutput{}, nil
					},
				}
			},
			messageID:   "draft-id",
			expectedErr: errors.New("1"),
		},
		{
			client: func(t *testing.T) platform.GetAndSendEmailAPI {
				t.Helper()
				return mockSendEmailAPI{
					mockGetItem: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
						return &dynamodb.GetItemOutput{
							Item: map[string]dynamodbTypes.AttributeValue{
								"MessageID":     &dynamodbTypes.AttributeValueMemberS{Value: "draft-id"},
								"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: "draft#2022-03"},
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
					},
					mockSendEmail: func(_ context.Context, _ *sesv2.SendEmailInput, _ ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
						return &sesv2.SendEmailOutput{
							MessageId: aws.String("newID"),
						}, nil
					},
					mockTransactWriteItem: func(_ context.Context, _ *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
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
		client            func(t *testing.T, email *Input) platform.SendEmailAPI
		email             *Input
		expectedMessageID string
		expectedErr       error
	}{
		{
			client: func(t *testing.T, email *Input) platform.SendEmailAPI {
				t.Helper()
				return mockSendEmailAPI{
					mockSendEmail: func(_ context.Context, params *sesv2.SendEmailInput, _ ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
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
			email: &Input{
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
			client: func(t *testing.T, _ *Input) platform.SendEmailAPI {
				t.Helper()
				return mockSendEmailAPI{
					mockSendEmail: func(_ context.Context, _ *sesv2.SendEmailInput, _ ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
						return &sesv2.SendEmailOutput{}, platform.ErrEmailIsNotDraft
					},
				}
			},
			email: &Input{
				From: []string{""},
			},
			expectedErr: platform.ErrEmailIsNotDraft,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Helper()
			ctx := context.TODO()
			messageID, err := sendEmailViaSES(ctx, test.client(t, test.email), test.email)
			assert.Equal(t, test.expectedMessageID, messageID)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}

func TestMarkEmailAsSent(t *testing.T) {
	tests := []struct {
		client       func(t *testing.T) platform.SendEmailAPI
		oldMessageID string
		email        *Input
		expectedErr  error
	}{
		{
			client: func(t *testing.T) platform.SendEmailAPI {
				t.Helper()
				return mockSendEmailAPI{
					mockTransactWriteItem: func(_ context.Context, params *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
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
			email: &Input{
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
			client: func(t *testing.T) platform.SendEmailAPI {
				t.Helper()
				return mockSendEmailAPI{
					mockTransactWriteItem: func(_ context.Context, _ *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
						return &dynamodb.TransactWriteItemsOutput{}, platform.ErrNotFound
					},
				}
			},
			email: &Input{
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
			expectedErr: platform.ErrNotFound,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Helper()
			ctx := context.TODO()
			err := markEmailAsSent(ctx, test.client(t), test.oldMessageID, test.email)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}

func TestBuildMIMEEmail(t *testing.T) {
	tests := []struct {
		input        *Input
		containLines []string
		noLines      []string
		expectedErr  error
	}{
		{
			input: &Input{
				Subject: "this is the subject",
				From:    []string{"Some One <someone@example.com>"},
				To:      []string{"To One <toone@example.com>"},
				ReplyTo: []string{"reply-to@example.com"},
				Text:    "this is the text",
				HTML:    "this is the html",
			},
			containLines: []string{
				"Subject: this is the subject",
				"From: \"Some One\" <someone@example.com>",
				"To: \"To One\" <toone@example.com>",
				"Reply-To: <reply-to@example.com>",
			},
			noLines: []string{
				"References: ",
				"In-Reply-To: ",
			},
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			email, err := buildMIMEEmail(test.input)
			assert.Equal(t, test.expectedErr, err)
			for _, line := range test.containLines {
				assert.Contains(t, string(email), line)
			}
			for _, line := range test.noLines {
				assert.NotContains(t, string(email), line)
			}
		})
	}
}

func TestConvertToMailAddresses(t *testing.T) {
	tests := []struct {
		input    []string
		expected []mail.Address
	}{
		{
			input: []string{"email@example.com"},
			expected: []mail.Address{
				{
					Name:    "",
					Address: "email@example.com",
				},
			},
		},
		{
			input: []string{"<email@example.com>"},
			expected: []mail.Address{
				{
					Name:    "",
					Address: "email@example.com",
				},
			},
		},
		{
			input: []string{"email@example.com", "First Last <foo@example.com>", "name <bar@example.com>"},
			expected: []mail.Address{
				{
					Name:    "",
					Address: "email@example.com",
				},
				{
					Name:    "First Last",
					Address: "foo@example.com",
				},
				{
					Name:    "name",
					Address: "bar@example.com",
				},
			},
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Helper()
			actual, err := convertToMailAddresses(test.input)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, actual)
		})
	}
}
