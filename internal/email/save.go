package email

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/api"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/types"
	"github.com/harryzcy/mailbox/internal/util/format"
)

// SaveInput represents the input of save method
type SaveInput struct {
	Input
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
//
// TODO: refactor this function
//
//gocyclo:ignore
func Save(ctx context.Context, client api.SaveAndSendEmailAPI, input SaveInput) (*SaveResult, error) {
	fmt.Println("save method started")
	if !strings.HasPrefix(input.MessageID, "draft-") {
		return nil, api.ErrEmailIsNotDraft
	}

	now := getUpdatedTime()
	typeYearMonth, err := format.TypeYearMonth(types.EmailTypeDraft, now)
	if err != nil {
		return nil, err
	}
	dateTime := format.DateTime(now)

	if (input.GenerateText == "on") || (input.GenerateText == "auto" && input.Text == "") {
		input.Text, err = generateText(input.HTML)
		if err != nil {
			return nil, err
		}
	}
	item := input.GenerateAttributes(typeYearMonth, dateTime)

	// The attributes ThreadID, InReplyTo, References are not included in the input,
	// but rather they are initialized when creating the draft email.
	// So we need to get the original values from DynamoDB, and keep them in the item.
	resp, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(env.TableName),
		Key: map[string]dynamodbTypes.AttributeValue{
			"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: input.MessageID},
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
			extraFields[key] = value.(*dynamodbTypes.AttributeValueMemberS).Value
		}
	}

	_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(env.TableName),
		Item:                item,
		ConditionExpression: aws.String("MessageID = :messageID"),
		ExpressionAttributeValues: map[string]dynamodbTypes.AttributeValue{
			":messageID": &dynamodbTypes.AttributeValueMemberS{Value: input.MessageID},
		},
	})
	if err != nil {
		if apiErr := new(dynamodbTypes.ConditionalCheckFailedException); errors.As(err, &apiErr) {
			return nil, api.ErrNotFound
		}
		if apiErr := new(dynamodbTypes.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return nil, api.ErrTooManyRequests
		}

		return nil, err
	}

	emailType := types.EmailTypeDraft
	messageID := input.MessageID
	if input.Send {
		email := &Input{
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
		if newMessageID, err = sendEmailViaSES(ctx, client, email); err != nil {
			return nil, err
		}
		email.MessageID = newMessageID

		if err = markEmailAsSent(ctx, client, messageID, email); err != nil {
			return nil, err
		}
		messageID = newMessageID
		emailType = types.EmailTypeSent
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
