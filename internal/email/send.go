package email

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sestypes "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/harryzcy/mailbox/internal/util/format"
)

// Send sends a draft email
func Send(ctx context.Context, api SendEmailAPI, messageID string) error {
	if !strings.HasPrefix(messageID, "draft-") {
		return ErrEmailIsNotDraft
	}

	email, err := Get(ctx, api, messageID)
	fmt.Println(email, err)
	if err != nil {
		return err
	}

	newMessageID, err := sendEmailViaSES(ctx, api, email)
	if err != nil {
		return err
	}

	err = markEmailAsSent(ctx, api, email.MessageID, EmailInput{
		MessageID: newMessageID,
		Subject:   email.Subject,
		From:      email.From,
		To:        email.To,
		Cc:        email.Cc,
		Bcc:       email.Bcc,
		ReplyTo:   email.ReplyTo,
		Text:      email.Text,
		HTML:      email.HTML,
	})
	if err != nil {
		return err
	}

	fmt.Println("send method finished successfully")
	return nil
}

func sendEmailViaSES(ctx context.Context, api SendEmailAPI, email *GetResult) (string, error) {
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

func markEmailAsSent(ctx context.Context, api SendEmailAPI, oldMessageID string, email EmailInput) error {
	now := getUpdatedTime()
	typeYearMonth, _ := format.FormatTypeYearMonth(EmailTypeSent, now)
	dateTime := format.FormatDateTime(now)

	item := email.GenerateAttributes(typeYearMonth, dateTime)

	_, err := api.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]dynamodbtypes.WriteRequest{
			tableName: {
				{
					DeleteRequest: &dynamodbtypes.DeleteRequest{
						Key: map[string]dynamodbtypes.AttributeValue{
							"MessageID": &dynamodbtypes.AttributeValueMemberS{Value: oldMessageID},
						},
					},
				},
				{
					PutRequest: &dynamodbtypes.PutRequest{
						Item: item,
					},
				},
			},
		},
		ReturnConsumedCapacity: dynamodbtypes.ReturnConsumedCapacityNone,
	})

	if err != nil {
		return err
	}
	return nil
}
