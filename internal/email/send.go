package email

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sestypes "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/harryzcy/mailbox/internal/util/format"
)

type SendResult struct {
	MessageID string
}

// Send sends a draft email
func Send(ctx context.Context, api GetAndSendEmailAPI, messageID string) (*SendResult, error) {
	if !strings.HasPrefix(messageID, "draft-") {
		return nil, ErrEmailIsNotDraft
	}

	resp, err := Get(ctx, api, messageID)
	if err != nil {
		return nil, err
	}

	email := &EmailInput{
		MessageID: messageID,
		Subject:   resp.Subject,
		From:      resp.From,
		To:        resp.To,
		Cc:        resp.Cc,
		Bcc:       resp.Bcc,
		ReplyTo:   resp.ReplyTo,
		Text:      resp.Text,
		HTML:      resp.HTML,
		ThreadID:  resp.ThreadID,
	}
	newMessageID, err := sendEmailViaSES(ctx, api, email)
	if err != nil {
		return nil, err
	}
	email.MessageID = newMessageID

	err = markEmailAsSent(ctx, api, messageID, email)
	if err != nil {
		return nil, err
	}

	fmt.Println("send method finished successfully")
	return &SendResult{
		MessageID: newMessageID,
	}, nil
}

func sendEmailViaSES(ctx context.Context, api SendEmailAPI, email *EmailInput) (string, error) {
	resp, err := api.SendEmail(ctx, &sesv2.SendEmailInput{
		Content: &sestypes.EmailContent{
			Simple: &sestypes.Message{
				Body: &sestypes.Body{
					Html: &sestypes.Content{
						Data:    aws.String(email.HTML),
						Charset: aws.String("UTF-8"),
					},
					Text: &sestypes.Content{
						Data:    aws.String(email.Text),
						Charset: aws.String("UTF-8"),
					},
				},
				Subject: &sestypes.Content{
					Data:    aws.String(email.Subject),
					Charset: aws.String("UTF-8"),
				},
			},
		},
		Destination: &sestypes.Destination{
			ToAddresses:  email.To,
			CcAddresses:  email.Cc,
			BccAddresses: email.Bcc,
		},
		FromEmailAddress: aws.String(email.From[0]),
		ReplyToAddresses: email.ReplyTo,
	})
	if err != nil {
		return "", err
	}

	return *resp.MessageId, nil
}

func markEmailAsSent(ctx context.Context, api SendEmailAPI, oldMessageID string, email *EmailInput) error {
	now := getUpdatedTime()
	typeYearMonth, _ := format.FormatTypeYearMonth(EmailTypeSent, now)
	dateTime := format.FormatDateTime(now)

	item := email.GenerateAttributes(typeYearMonth, dateTime)

	_, err := api.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []dynamodbtypes.TransactWriteItem{
			{
				Delete: &dynamodbtypes.Delete{
					TableName: aws.String(tableName),
					Key: map[string]dynamodbtypes.AttributeValue{
						"MessageID": &dynamodbtypes.AttributeValueMemberS{Value: oldMessageID},
					},
				},
			},
			{
				Put: &dynamodbtypes.Put{
					TableName: aws.String(tableName),
					Item:      item,
				},
			},
		},
	})

	if err != nil {
		if apiErr := new(dynamodbtypes.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return ErrTooManyRequests
		}
		return err
	}
	return nil
}
