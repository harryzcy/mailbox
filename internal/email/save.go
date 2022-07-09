package email

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/util/format"
)

// SaveInput represents the input of save method
type SaveInput struct {
	EmailInput
	GenerateText string `json:"generateText"` // on, off, or auto (default)
	Send         bool   `json:"send"`         // send email immediately
}

// SaveResult represents the result of save method
type SaveResult struct {
	TimeIndex
	Subject string   `json:"subject"`
	From    []string `json:"from"`
	To      []string `json:"to"`
	Cc      []string `json:"cc"`
	Bcc     []string `json:"bcc"`
	ReplyTo []string `json:"replyTo"`
	Text    string   `json:"text"`
	HTML    string   `json:"html"`
}

var getUpdatedTime = func() time.Time {
	return time.Now().UTC()
}

// Save puts an email as draft in DynamoDB
func Save(ctx context.Context, api SaveAndSendEmailAPI, input SaveInput) (*SaveResult, error) {
	if !strings.HasPrefix(input.MessageID, "draft-") {
		return nil, ErrEmailIsNotDraft
	}

	now := getUpdatedTime()
	typeYearMonth, _ := format.FormatTypeYearMonth(EmailTypeDraft, now)
	dateTime := format.FormatDateTime(now)

	if (input.GenerateText == "on") || (input.GenerateText == "auto" && input.Text == "") {
		var err error
		input.Text, err = generateText(input.HTML)
		if err != nil {
			return nil, err
		}
	}
	item := input.GenerateAttributes(typeYearMonth, dateTime)

	_, err := api.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(tableName),
		Item:                item,
		ConditionExpression: aws.String("MessageID = :messageID"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":messageID": &types.AttributeValueMemberS{Value: input.MessageID},
		},
	})
	if err != nil {
		var condFailedErr *types.ConditionalCheckFailedException
		if errors.As(err, &condFailedErr) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	emailType := EmailTypeDraft
	messageID := input.MessageID
	if input.Send {
		email := &EmailInput{
			MessageID: messageID,
			Subject:   input.Subject,
			From:      input.From,
			To:        input.To,
			Cc:        input.Cc,
			Bcc:       input.Bcc,
			ReplyTo:   input.ReplyTo,
			Text:      input.Text,
			HTML:      input.HTML,
		}

		var newMessageID string
		if newMessageID, err = sendEmailViaSES(ctx, api, email); err != nil {
			return nil, err
		}
		email.MessageID = newMessageID

		if err = markEmailAsSent(ctx, api, messageID, email); err != nil {
			return nil, err
		}
		messageID = newMessageID
		emailType = EmailTypeSent
	}

	result := &SaveResult{
		TimeIndex: TimeIndex{
			MessageID:   messageID,
			Type:        emailType,
			TimeUpdated: now.Format(time.RFC3339),
		},
		Subject: input.Subject,
		From:    input.From,
		To:      input.To,
		Cc:      input.Cc,
		Bcc:     input.Bcc,
		ReplyTo: input.ReplyTo,
		Text:    input.Text,
		HTML:    input.HTML,
	}

	fmt.Println("save method finished successfully")
	return result, nil
}
