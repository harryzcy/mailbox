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
	Subject  string   `json:"subject"`
	From     []string `json:"from"`
	To       []string `json:"to"`
	Cc       []string `json:"cc"`
	Bcc      []string `json:"bcc"`
	ReplyTo  []string `json:"replyTo"`
	Text     string   `json:"text"`
	HTML     string   `json:"html"`
	ThreadID string   `json:"threadID,omitempty"`
}

var getUpdatedTime = func() time.Time {
	return time.Now().UTC()
}

// Save puts an email as draft in DynamoDB
func Save(ctx context.Context, api SaveAndSendEmailAPI, input SaveInput) (*SaveResult, error) {
	fmt.Println("save method started")
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

	// The attributes ThreadID, InReplyTo, References are not included in the input,
	// but rather they are initialized when creating the draft email.
	// So we need to get the original values from DynamoDB, and keep them in the item.
	resp, err := api.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"MessageID": &types.AttributeValueMemberS{Value: input.MessageID},
		},
	})
	if err != nil {
		return nil, err
	}

	// ThreadID, InReplyTo, References are included only if they exist
	var extraFields = map[string]string{
		"ThreadID":   "",
		"InReplyTo":  "",
		"References": "",
	}
	for key := range extraFields {
		if value, ok := resp.Item[key]; ok {
			item[key] = value // keep the original value
			extraFields[key] = value.(*types.AttributeValueMemberS).Value
		}
	}

	_, err = api.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(tableName),
		Item:                item,
		ConditionExpression: aws.String("MessageID = :messageID"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":messageID": &types.AttributeValueMemberS{Value: input.MessageID},
		},
	})
	if err != nil {
		if apiErr := new(types.ConditionalCheckFailedException); errors.As(err, &apiErr) {
			return nil, ErrNotFound
		}
		if apiErr := new(types.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return nil, ErrTooManyRequests
		}

		return nil, err
	}

	emailType := EmailTypeDraft
	messageID := input.MessageID
	if input.Send {
		email := &EmailInput{
			MessageID:  messageID,
			Subject:    input.Subject,
			From:       input.From,
			To:         input.To,
			Cc:         input.Cc,
			Bcc:        input.Bcc,
			ReplyTo:    input.ReplyTo,
			Text:       input.Text,
			HTML:       input.HTML,
			ThreadID:   extraFields["ThreadID"],
			InReplyTo:  extraFields["InReplyTo"],
			References: extraFields["References"],
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
		Subject:  input.Subject,
		From:     input.From,
		To:       input.To,
		Cc:       input.Cc,
		Bcc:      input.Bcc,
		ReplyTo:  input.ReplyTo,
		Text:     input.Text,
		HTML:     input.HTML,
		ThreadID: extraFields["ThreadID"],
	}

	fmt.Println("save method finished successfully")
	return result, nil
}
