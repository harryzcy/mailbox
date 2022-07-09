package email

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"
	"github.com/harryzcy/mailbox/internal/util/htmlutil"
)

// CreateInput represents the input of create method
type CreateInput struct {
	EmailInput
	GenerateText string `json:"generateText"` // on, off, or auto (default)
	Send         bool   `json:"send"`         // send email immediately
}

// CreateResult represents the result of create method
type CreateResult struct {
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

func generateMessageID() string {
	rawID := uuid.New()
	messageID := "draft-" + strings.ReplaceAll(rawID.String(), "-", "")
	return messageID
}

var generateText = htmlutil.GenerateText

// Create adds an email as draft in DynamoDB
func Create(ctx context.Context, api SaveAndSendEmailAPI, input CreateInput) (*CreateResult, error) {
	messageID := generateMessageID()
	now := getUpdatedTime()
	typeYearMonth := EmailTypeDraft + "#" + now.Format("2006-01")
	dateTime := now.Format("02-15:04:05")

	input.MessageID = messageID
	if (input.GenerateText == "on") || (input.GenerateText == "auto" && input.Text == "") {
		var err error
		input.Text, err = generateText(input.HTML)
		fmt.Println(err)
		if err != nil {
			return nil, err
		}
	}

	item := input.GenerateAttributes(typeYearMonth, dateTime)

	_, err := api.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	})
	if err != nil {
		return nil, err
	}

	emailType := EmailTypeDraft
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

	result := &CreateResult{
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

	fmt.Println("create method finished successfully")
	return result, nil
}
